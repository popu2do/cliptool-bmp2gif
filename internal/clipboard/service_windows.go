package clipboard

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"cliptool/internal/applog"
	"cliptool/internal/core"

	"golang.org/x/sys/windows"
)

const (
	maxRetries    = 3
	maxPathLength = 32767
	cfHDrop       = 15
)

var (
	user32            = windows.NewLazySystemDLL("user32.dll")
	openClipboard     = user32.NewProc("OpenClipboard")
	closeClipboard    = user32.NewProc("CloseClipboard")
	getClipboardData  = user32.NewProc("GetClipboardData")
	getSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	dragQueryFileW    = shell32.NewProc("DragQueryFileW")
)

type Service struct {
	tempDir            string
	mu                 sync.Mutex
	lastLoggedSequence uint32
	hasLoggedSequence  bool
}

func NewService(tempDir string) *Service {
	return &Service{
		tempDir: tempDir,
	}
}

func (s *Service) ReadImageFiles() ([]string, string) {
	imageFiles := make([]string, 0)
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
		applog.Errorf("读取剪切板失败: 多次尝试后仍无法打开")
		return nil, ""
	}
	defer closeClipboard.Call()

	hdropHandle, _, lastErr := getClipboardData.Call(uintptr(cfHDrop))
	if hdropHandle == 0 {
		if verbose {
			applog.Debugf("剪切板没有 CF_HDROP 文件列表: err=%v sequence=%d", lastErr, sequenceNumber)
		}
		return nil, ""
	}

	fileCount, _, lastErr := dragQueryFileW.Call(hdropHandle, 0xFFFFFFFF, 0, 0)
	if fileCount == 0 {
		if verbose {
			applog.Debugf("剪切板 CF_HDROP 文件数为 0: err=%v sequence=%d", lastErr, sequenceNumber)
		}
		return nil, ""
	}
	if verbose {
		applog.Debugf("剪切板文件数量: count=%d sequence=%d", fileCount, sequenceNumber)
	}

	for fileIndex := uint32(0); fileIndex < uint32(fileCount); fileIndex++ {
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
		supported := core.IsSupportedImage(filePath)
		if verbose {
			logClipboardFile(fileIndex, filePath, supported)
		}
		if supported {
			imageFiles = append(imageFiles, filePath)
		}
	}

	contentHash := calcHash(imageFiles)
	if verbose {
		applog.Debugf("读取剪切板完成: supported=%d hash=%q sequence=%d", len(imageFiles), contentHash, sequenceNumber)
	}
	return imageFiles, contentHash
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
