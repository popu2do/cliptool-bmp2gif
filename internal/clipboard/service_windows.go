package clipboard

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

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
	tempDir string
}

func NewService(tempDir string) *Service {
	return &Service{
		tempDir: tempDir,
	}
}

func (s *Service) ReadImageFiles() ([]string, string) {
	imageFiles := make([]string, 0)

	clipboardOpened := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		ret, _, _ := openClipboard.Call(0)
		if ret != 0 {
			clipboardOpened = true
			break
		}
		time.Sleep(time.Millisecond * time.Duration(10<<uint(attempt)))
	}

	if !clipboardOpened {
		return nil, ""
	}
	defer closeClipboard.Call()

	hdropHandle, _, _ := getClipboardData.Call(uintptr(cfHDrop))
	if hdropHandle == 0 {
		return nil, ""
	}

	fileCount, _, _ := dragQueryFileW.Call(hdropHandle, 0xFFFFFFFF, 0, 0)
	if fileCount == 0 {
		return nil, ""
	}

	for fileIndex := uint32(0); fileIndex < uint32(fileCount); fileIndex++ {
		pathLength, _, _ := dragQueryFileW.Call(hdropHandle, uintptr(fileIndex), 0, 0)
		if pathLength == 0 || pathLength > maxPathLength {
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
		if core.IsSupportedImage(filePath) {
			imageFiles = append(imageFiles, filePath)
		}
	}

	return imageFiles, calcHash(imageFiles, clipboardSequenceNumber())
}

func (s *Service) WriteGIF(gifData []byte) error {
	if err := os.MkdirAll(s.tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	tempFilePath := filepath.Join(s.tempDir, fmt.Sprintf("temp_clipboard_%s.gif", timestamp))

	absPath, err := filepath.Abs(tempFilePath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	if err := os.WriteFile(tempFilePath, gifData, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	escapedPath := strings.ReplaceAll(absPath, "'", "''")
	command := fmt.Sprintf("Set-Clipboard -LiteralPath '%s'", escapedPath)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PowerShell 设置剪贴板失败: %v，输出: %s", err, string(output))
	}

	return nil
}

func (s *Service) ClearTempDirectory() {
	if _, err := os.Stat(s.tempDir); err == nil {
		_ = os.RemoveAll(s.tempDir)
	}
	_ = os.MkdirAll(s.tempDir, 0755)
}

func clipboardSequenceNumber() uint32 {
	sequenceNumber, _, _ := getSequenceNumber.Call()
	return uint32(sequenceNumber)
}

func calcHash(files []string, sequenceNumber uint32) string {
	if len(files) == 0 {
		return ""
	}

	hasher := md5.New()
	hasher.Write([]byte(fmt.Sprintf("%d", sequenceNumber)))
	for _, filePath := range files {
		hasher.Write([]byte(filePath))
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}
