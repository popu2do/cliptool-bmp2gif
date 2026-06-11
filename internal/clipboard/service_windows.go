package clipboard

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"cliptool/internal/applog"
	"cliptool/internal/core"

	"golang.org/x/sys/windows"
)

const (
	maxRetries       = 4
	maxPathLength    = 32767
	cfHDrop          = 15
	dvAspectContent  = 1
	sigdnFileSysPath = 0x80058000
	tymedHGlobal     = 1
)

var (
	user32                   = windows.NewLazySystemDLL("user32.dll")
	openClipboard            = user32.NewProc("OpenClipboard")
	closeClipboard           = user32.NewProc("CloseClipboard")
	enumFormats              = user32.NewProc("EnumClipboardFormats")
	getClipboardData         = user32.NewProc("GetClipboardData")
	getFormatName            = user32.NewProc("GetClipboardFormatNameW")
	getOpenClipboardWindow   = user32.NewProc("GetOpenClipboardWindow")
	getWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	isFormatAvailable        = user32.NewProc("IsClipboardFormatAvailable")
	registerFormat           = user32.NewProc("RegisterClipboardFormatW")
	getSequenceNumber        = user32.NewProc("GetClipboardSequenceNumber")
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	globalLock               = kernel32.NewProc("GlobalLock")
	globalUnlock             = kernel32.NewProc("GlobalUnlock")
	globalSize               = kernel32.NewProc("GlobalSize")
	ole32                    = windows.NewLazySystemDLL("ole32.dll")
	oleInitialize            = ole32.NewProc("OleInitialize")
	oleUninitialize          = ole32.NewProc("OleUninitialize")
	oleGetClipboard          = ole32.NewProc("OleGetClipboard")
	releaseStgMedium         = ole32.NewProc("ReleaseStgMedium")
	coTaskMemFree            = ole32.NewProc("CoTaskMemFree")
	shell32                  = windows.NewLazySystemDLL("shell32.dll")
	dragQueryFileW           = shell32.NewProc("DragQueryFileW")
	ilCombine                = shell32.NewProc("ILCombine")
	ilFree                   = shell32.NewProc("ILFree")
	shGetNameFromIDList      = shell32.NewProc("SHGetNameFromIDList")
)

type formatEtc struct {
	cfFormat uint16
	ptd      uintptr
	dwAspect uint32
	lindex   int32
	tymed    uint32
}

type stgMedium struct {
	tymed          uint32
	unionMember    uintptr
	pUnkForRelease uintptr
}

type iDataObject struct {
	lpVtbl *iDataObjectVtbl
}

type iDataObjectVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
	getData        uintptr
}

type Service struct {
	tempDir            string
	mu                 sync.Mutex
	lastLoggedSequence uint32
	hasLoggedSequence  bool
}

type ReadResult struct {
	ImageFiles          []string
	ContentHash         string
	TotalFiles          int
	UnsupportedFiles    []UnsupportedFile
	NonFileContent      bool
	FileListUnreadable  bool
	ClipboardOpenFailed bool
}

type UnsupportedFile struct {
	Path  string
	Name  string
	Ext   string
	Size  int64
	IsDir bool
}

func NewService(tempDir string) *Service {
	return &Service{
		tempDir: tempDir,
	}
}

func (s *Service) ReadImageFiles() ReadResult {
	sequenceNumber := clipboardSequenceNumber()
	verbose := s.shouldLogSequence(sequenceNumber)
	if verbose {
		applog.Debugf("读取剪切板开始: sequence=%d", sequenceNumber)
	}

	clipboardOpened := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		ret, _, lastErr := openClipboard.Call(0)
		if ret != 0 {
			clipboardOpened = true
			if verbose {
				applog.Debugf("打开剪切板成功: attempt=%d", attempt+1)
			}
			break
		}
		if verbose {
			applog.Warnf("打开剪切板失败: attempt=%d err=%v", attempt+1, lastErr)
		}
		time.Sleep(time.Millisecond * time.Duration(10<<uint(attempt)))
	}

	if !clipboardOpened {
		if verbose {
			applog.Warnf("读取剪切板失败: 多次尝试后仍无法打开 sequence=%d owner=%s", sequenceNumber, clipboardOwnerSummary())
		}
		return clipboardOpenFailedReadResult(sequenceNumber)
	}
	clipboardCurrentlyOpen := true
	defer func() {
		if clipboardCurrentlyOpen {
			closeClipboard.Call()
		}
	}()

	hdropHandle, _, lastErr := getClipboardData.Call(uintptr(cfHDrop))
	if hdropHandle == 0 {
		fileNamePaths := readClipboardFileNamePaths()
		if len(fileNamePaths) > 0 {
			if verbose {
				applog.Debugf("通过 FileName/FileNameW 读取剪切板文件路径成功: count=%d sequence=%d", len(fileNamePaths), sequenceNumber)
			}
			return buildReadResult(fileNamePaths, sequenceNumber, verbose)
		}

		shellIDListPaths, shellIDListErr := readShellIDListFilePaths()
		if len(shellIDListPaths) > 0 {
			if verbose {
				applog.Debugf("通过 Shell IDList Array 读取剪切板文件列表成功: count=%d sequence=%d", len(shellIDListPaths), sequenceNumber)
			}
			return buildReadResult(shellIDListPaths, sequenceNumber, verbose)
		}

		if clipboardFormatAvailable(cfHDrop) {
			formats := clipboardFormatSummary()
			closeClipboard.Call()
			clipboardCurrentlyOpen = false

			files, err := readOleClipboardFilePaths()
			if err == nil && len(files) > 0 {
				if verbose {
					applog.Debugf("通过 OLE IDataObject 读取剪切板文件列表成功: count=%d sequence=%d", len(files), sequenceNumber)
				}
				return buildReadResult(files, sequenceNumber, verbose)
			}
			if verbose {
				applog.Debugf("剪切板声明了 CF_HDROP 但文件列表不可读: err=%v shellIDListErr=%v oleErr=%v sequence=%d formats=%s", lastErr, shellIDListErr, err, sequenceNumber, formats)
			}
			return unreadableFileListReadResult(sequenceNumber)
		}
		if clipboardRegisteredFormatAvailable("FileNameW") || clipboardRegisteredFormatAvailable("FileName") {
			if verbose {
				applog.Debugf("剪切板声明了 FileName/FileNameW 但路径不可读: err=%v sequence=%d formats=%s", lastErr, sequenceNumber, clipboardFormatSummary())
			}
			return unreadableFileListReadResult(sequenceNumber)
		}
		if verbose {
			applog.Debugf("剪切板没有 CF_HDROP 文件列表: err=%v sequence=%d", lastErr, sequenceNumber)
		}
		return nonFileReadResult(sequenceNumber)
	}

	fileCount, _, lastErr := dragQueryFileW.Call(hdropHandle, 0xFFFFFFFF, 0, 0)
	if fileCount == 0 {
		shellIDListPaths, shellIDListErr := readShellIDListFilePaths()
		if len(shellIDListPaths) > 0 {
			if verbose {
				applog.Debugf("通过 Shell IDList Array 读取剪切板文件列表成功: count=%d sequence=%d", len(shellIDListPaths), sequenceNumber)
			}
			return buildReadResult(shellIDListPaths, sequenceNumber, verbose)
		}
		if verbose {
			applog.Debugf("剪切板 CF_HDROP 文件数为 0: err=%v shellIDListErr=%v sequence=%d", lastErr, shellIDListErr, sequenceNumber)
		}
		return nonFileReadResult(sequenceNumber)
	}

	files := readDropFilePaths(hdropHandle, uint32(fileCount))
	if len(files) == 0 {
		shellIDListPaths, shellIDListErr := readShellIDListFilePaths()
		if len(shellIDListPaths) > 0 {
			if verbose {
				applog.Debugf("通过 Shell IDList Array 读取剪切板文件列表成功: count=%d sequence=%d", len(shellIDListPaths), sequenceNumber)
			}
			return buildReadResult(shellIDListPaths, sequenceNumber, verbose)
		}
		if verbose {
			applog.Debugf("剪切板 CF_HDROP 路径为空且 Shell IDList Array 不可读: shellIDListErr=%v sequence=%d", shellIDListErr, sequenceNumber)
		}
		return nonFileReadResult(sequenceNumber)
	}
	if verbose {
		applog.Debugf("剪切板文件数量: count=%d sequence=%d", len(files), sequenceNumber)
	}

	return buildReadResult(files, sequenceNumber, verbose)
}

func readDropFilePaths(hdropHandle uintptr, fileCount uint32) []string {
	files := make([]string, 0, fileCount)
	for fileIndex := uint32(0); fileIndex < fileCount; fileIndex++ {
		pathLength, _, _ := dragQueryFileW.Call(hdropHandle, uintptr(fileIndex), 0, 0)
		if pathLength == 0 || pathLength > maxPathLength {
			applog.Warnf("跳过异常剪切板路径长度: index=%d length=%d", fileIndex, pathLength)
			continue
		}

		pathBuffer := make([]uint16, pathLength+1)
		dragQueryFileW.Call(
			hdropHandle,
			uintptr(fileIndex),
			uintptr(unsafe.Pointer(&pathBuffer[0])),
			uintptr(len(pathBuffer)),
		)

		filePath := windows.UTF16ToString(pathBuffer)
		files = append(files, filePath)
	}
	return files
}

func readShellIDListFilePaths() ([]string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := oleInitialize.Call(0)
	if failedHRESULT(hr) {
		return nil, fmt.Errorf("OleInitialize for Shell IDList Array failed: 0x%x", hr)
	}
	defer oleUninitialize.Call()

	format := registeredClipboardFormat("Shell IDList Array")
	if format == 0 {
		return nil, fmt.Errorf("Shell IDList Array format is unavailable")
	}
	if !clipboardFormatAvailable(format) {
		return nil, fmt.Errorf("Shell IDList Array is not present")
	}

	handle, _, lastErr := getClipboardData.Call(uintptr(format))
	if handle == 0 {
		return nil, fmt.Errorf("GetClipboardData(Shell IDList Array) failed: %v", lastErr)
	}

	pointer, _, lastErr := globalLock.Call(handle)
	if pointer == 0 {
		return nil, fmt.Errorf("GlobalLock(Shell IDList Array) failed: %v", lastErr)
	}
	defer globalUnlock.Call(handle)

	size, _, _ := globalSize.Call(handle)
	if size == 0 || size > 16*1024*1024 {
		return nil, fmt.Errorf("invalid Shell IDList Array size: %d", size)
	}

	data := unsafe.Slice((*byte)(unsafe.Pointer(pointer)), int(size))
	offsets, err := parseCIDAOffsets(data)
	if err != nil {
		return nil, err
	}
	if len(offsets) < 2 {
		return nil, fmt.Errorf("Shell IDList Array contains no child PIDLs")
	}

	parentPIDL := pointer + uintptr(offsets[0])
	files := make([]string, 0, len(offsets)-1)
	for _, childOffset := range offsets[1:] {
		childPIDL := pointer + uintptr(childOffset)
		fullPIDL, _, _ := ilCombine.Call(parentPIDL, childPIDL)
		if fullPIDL == 0 {
			applog.Warnf("Shell IDList Array PIDL 合并失败: childOffset=%d", childOffset)
			continue
		}

		filePath, err := shellIDListPath(fullPIDL)
		ilFree.Call(fullPIDL)
		if err != nil {
			applog.Warnf("Shell IDList Array PIDL 转路径失败: childOffset=%d err=%v", childOffset, err)
			continue
		}
		files = append(files, filePath)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("Shell IDList Array did not resolve any file paths")
	}
	return files, nil
}

func parseCIDAOffsets(data []byte) ([]uint32, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("Shell IDList Array is too short: %d", len(data))
	}

	childCount := binary.LittleEndian.Uint32(data[:4])
	if childCount > 10000 {
		return nil, fmt.Errorf("Shell IDList Array child count is too large: %d", childCount)
	}

	offsetCount := childCount + 1
	offsetTableBytes := 4 + int(offsetCount)*4
	if offsetTableBytes > len(data) {
		return nil, fmt.Errorf("Shell IDList Array offset table is truncated: count=%d size=%d", childCount, len(data))
	}

	offsets := make([]uint32, 0, offsetCount)
	for i := uint32(0); i < offsetCount; i++ {
		offset := binary.LittleEndian.Uint32(data[4+i*4:])
		if offset >= uint32(len(data)) {
			return nil, fmt.Errorf("Shell IDList Array offset is out of range: index=%d offset=%d size=%d", i, offset, len(data))
		}
		offsets = append(offsets, offset)
	}
	return offsets, nil
}

func shellIDListPath(pidl uintptr) (string, error) {
	var pathPointer uintptr
	hr, _, _ := shGetNameFromIDList.Call(
		pidl,
		uintptr(sigdnFileSysPath),
		uintptr(unsafe.Pointer(&pathPointer)),
	)
	if failedHRESULT(hr) {
		return "", fmt.Errorf("SHGetNameFromIDList(SIGDN_FILESYSPATH) failed: 0x%x", hr)
	}
	if pathPointer == 0 {
		return "", fmt.Errorf("SHGetNameFromIDList(SIGDN_FILESYSPATH) returned nil")
	}
	defer coTaskMemFree.Call(pathPointer)

	filePath := strings.TrimSpace(windows.UTF16PtrToString((*uint16)(unsafe.Pointer(pathPointer))))
	if filePath == "" {
		return "", fmt.Errorf("SHGetNameFromIDList(SIGDN_FILESYSPATH) returned empty path")
	}
	return filePath, nil
}

func nonFileReadResult(sequenceNumber uint32) ReadResult {
	return ReadResult{
		ContentHash:    fmt.Sprintf("non-file:%d", sequenceNumber),
		NonFileContent: true,
	}
}

func unreadableFileListReadResult(sequenceNumber uint32) ReadResult {
	return ReadResult{
		ContentHash:        fmt.Sprintf("unreadable-file-list:%d", sequenceNumber),
		FileListUnreadable: true,
	}
}

func clipboardOpenFailedReadResult(sequenceNumber uint32) ReadResult {
	return ReadResult{
		ContentHash:         fmt.Sprintf("clipboard-open-failed:%d", sequenceNumber),
		ClipboardOpenFailed: true,
	}
}

func clipboardFormatAvailable(format uint32) bool {
	available, _, _ := isFormatAvailable.Call(uintptr(format))
	return available != 0
}

func clipboardOwnerSummary() string {
	windowHandle, _, _ := getOpenClipboardWindow.Call()
	if windowHandle == 0 {
		return "hwnd=0 pid=0"
	}
	var processID uint32
	getWindowThreadProcessID.Call(windowHandle, uintptr(unsafe.Pointer(&processID)))
	return fmt.Sprintf("hwnd=0x%x pid=%d", windowHandle, processID)
}

func clipboardFormatSummary() string {
	formats := make([]string, 0)
	var current uintptr
	for {
		next, _, _ := enumFormats.Call(current)
		if next == 0 {
			break
		}
		formats = append(formats, clipboardFormatName(uint32(next)))
		current = next
	}
	if len(formats) == 0 {
		return "<none>"
	}
	return strings.Join(formats, ",")
}

func clipboardFormatName(format uint32) string {
	switch format {
	case 2:
		return "CF_BITMAP"
	case 13:
		return "CF_UNICODETEXT"
	case cfHDrop:
		return "CF_HDROP"
	case 17:
		return "CF_DIBV5"
	}

	buffer := make([]uint16, 128)
	nameLength, _, _ := getFormatName.Call(
		uintptr(format),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if nameLength > 0 {
		return windows.UTF16ToString(buffer[:nameLength])
	}
	return fmt.Sprintf("format:%d", format)
}

func readClipboardFileNamePaths() []string {
	if paths := readClipboardRegisteredFileName("FileNameW", true); len(paths) > 0 {
		return paths
	}
	return readClipboardRegisteredFileName("FileName", false)
}

func readClipboardRegisteredFileName(formatName string, unicode bool) []string {
	format := registeredClipboardFormat(formatName)
	if format == 0 {
		return nil
	}
	if !clipboardFormatAvailable(format) {
		return nil
	}

	handle, _, _ := getClipboardData.Call(uintptr(format))
	if handle == 0 {
		return nil
	}

	pointer, _, _ := globalLock.Call(handle)
	if pointer == 0 {
		return nil
	}
	defer globalUnlock.Call(handle)

	var filePath string
	if unicode {
		filePath = windows.UTF16PtrToString((*uint16)(unsafe.Pointer(pointer)))
	} else {
		filePath = windows.BytePtrToString((*byte)(unsafe.Pointer(pointer)))
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil
	}
	if _, err := os.Stat(filePath); err != nil {
		return nil
	}
	return []string{filePath}
}

func clipboardRegisteredFormatAvailable(name string) bool {
	format := registeredClipboardFormat(name)
	return format != 0 && clipboardFormatAvailable(format)
}

func registeredClipboardFormat(name string) uint32 {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	format, _, _ := registerFormat.Call(uintptr(unsafe.Pointer(namePtr)))
	return uint32(format)
}

func readOleClipboardFilePaths() ([]string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := oleInitialize.Call(0)
	if failedHRESULT(hr) {
		return nil, fmt.Errorf("OleInitialize failed: 0x%x", hr)
	}
	defer oleUninitialize.Call()

	var dataObject *iDataObject
	hr, _, _ = oleGetClipboard.Call(uintptr(unsafe.Pointer(&dataObject)))
	if failedHRESULT(hr) {
		return nil, fmt.Errorf("OleGetClipboard failed: 0x%x", hr)
	}
	if dataObject == nil {
		return nil, fmt.Errorf("OleGetClipboard returned nil data object")
	}
	defer dataObject.Release()

	format := formatEtc{
		cfFormat: cfHDrop,
		dwAspect: dvAspectContent,
		lindex:   -1,
		tymed:    tymedHGlobal,
	}
	var medium stgMedium
	hr = dataObject.GetData(&format, &medium)
	if failedHRESULT(hr) {
		return nil, fmt.Errorf("IDataObject.GetData(CF_HDROP) failed: 0x%x", hr)
	}
	defer releaseStgMedium.Call(uintptr(unsafe.Pointer(&medium)))

	fileCount, _, lastErr := dragQueryFileW.Call(medium.unionMember, 0xFFFFFFFF, 0, 0)
	if fileCount == 0 {
		return nil, fmt.Errorf("OLE CF_HDROP file count is 0: %v", lastErr)
	}

	return readDropFilePaths(medium.unionMember, uint32(fileCount)), nil
}

func failedHRESULT(hr uintptr) bool {
	return int32(hr) < 0
}

func (obj *iDataObject) GetData(format *formatEtc, medium *stgMedium) uintptr {
	hr, _, _ := syscall.SyscallN(
		obj.lpVtbl.getData,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(format)),
		uintptr(unsafe.Pointer(medium)),
	)
	return hr
}

func (obj *iDataObject) Release() {
	syscall.SyscallN(obj.lpVtbl.release, uintptr(unsafe.Pointer(obj)))
}

func buildReadResult(files []string, sequenceNumber uint32, verbose bool) ReadResult {
	imageFiles := make([]string, 0)
	allFiles := make([]string, 0, len(files))
	unsupportedFiles := make([]UnsupportedFile, 0)

	for fileIndex, filePath := range files {
		allFiles = append(allFiles, filePath)
		supported := core.IsSupportedImage(filePath)
		if verbose {
			logClipboardFile(uint32(fileIndex), filePath, supported)
		}
		if supported {
			imageFiles = append(imageFiles, filePath)
		} else if strings.EqualFold(filepath.Ext(filePath), ".gif") {
			continue
		} else if info, err := os.Stat(filePath); err == nil {
			unsupportedFiles = append(unsupportedFiles, unsupportedFileFromInfo(filePath, info))
		}
	}

	contentHash := calcHash(allFiles)
	if verbose {
		applog.Debugf("读取剪切板完成: total=%d supported=%d unsupported=%d hash=%q sequence=%d", len(allFiles), len(imageFiles), len(unsupportedFiles), contentHash, sequenceNumber)
	}
	return ReadResult{
		ImageFiles:       imageFiles,
		ContentHash:      contentHash,
		TotalFiles:       len(allFiles),
		UnsupportedFiles: unsupportedFiles,
	}
}

func unsupportedFileFromInfo(filePath string, info os.FileInfo) UnsupportedFile {
	return UnsupportedFile{
		Path:  filePath,
		Name:  filepath.Base(filePath),
		Ext:   filepath.Ext(filePath),
		Size:  info.Size(),
		IsDir: info.IsDir(),
	}
}

func (s *Service) WriteGIF(gifData []byte) error {
	applog.Debugf("写入 GIF 到剪切板开始: bytes=%d tempDir=%q", len(gifData), s.tempDir)
	if err := os.MkdirAll(s.tempDir, 0755); err != nil {
		applog.Errorf("创建临时目录失败: dir=%q err=%v", s.tempDir, err)
		return fmt.Errorf("创建临时目录失败: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	tempFilePath := filepath.Join(s.tempDir, fmt.Sprintf("temp_clipboard_%s.gif", timestamp))

	absPath, err := filepath.Abs(tempFilePath)
	if err != nil {
		applog.Errorf("获取 GIF 临时文件绝对路径失败: path=%q err=%v", tempFilePath, err)
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	if err := os.WriteFile(tempFilePath, gifData, 0644); err != nil {
		applog.Errorf("写入 GIF 临时文件失败: path=%q bytes=%d err=%v", tempFilePath, len(gifData), err)
		return fmt.Errorf("写入临时文件失败: %v", err)
	}
	applog.Debugf("GIF 临时文件已写入: path=%q abs=%q bytes=%d", tempFilePath, absPath, len(gifData))

	escapedPath := strings.ReplaceAll(absPath, "'", "''")
	command := fmt.Sprintf("Set-Clipboard -LiteralPath '%s'", escapedPath)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		applog.Errorf("PowerShell 设置 GIF 剪切板失败: path=%q err=%v output=%q", absPath, err, string(output))
		return fmt.Errorf("PowerShell 设置剪贴板失败: %v，输出: %s", err, string(output))
	}

	applog.Infof("GIF 已写入剪切板: path=%q bytes=%d", absPath, len(gifData))
	return nil
}

func (s *Service) ClearTempDirectory() {
	applog.Debugf("清理临时目录: dir=%q", s.tempDir)
	if _, err := os.Stat(s.tempDir); err == nil {
		_ = os.RemoveAll(s.tempDir)
	}
	if err := os.MkdirAll(s.tempDir, 0755); err != nil {
		applog.Errorf("创建临时目录失败: dir=%q err=%v", s.tempDir, err)
	}
}

func clipboardSequenceNumber() uint32 {
	sequenceNumber, _, _ := getSequenceNumber.Call()
	return uint32(sequenceNumber)
}

func calcHash(files []string) string {
	if len(files) == 0 {
		return ""
	}

	hasher := md5.New()
	for _, filePath := range files {
		hasher.Write([]byte(strings.ToLower(filepath.Clean(filePath))))
		hasher.Write([]byte{0})
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func logClipboardFile(index uint32, filePath string, supported bool) {
	info, err := os.Stat(filePath)
	if err != nil {
		applog.Warnf("剪切板文件检查失败: index=%d path=%q ext=%q supported=%t err=%v", index, filePath, filepath.Ext(filePath), supported, err)
		return
	}
	applog.Debugf("剪切板文件: index=%d path=%q ext=%q size=%d modified=%s supported=%t", index, filePath, filepath.Ext(filePath), info.Size(), info.ModTime().Format(time.RFC3339), supported)
}

func (s *Service) shouldLogSequence(sequenceNumber uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasLoggedSequence && sequenceNumber == s.lastLoggedSequence {
		return false
	}
	s.lastLoggedSequence = sequenceNumber
	s.hasLoggedSequence = true
	return true
}
