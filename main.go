// ClipTool - 剪贴板图片转GIF工具
//
// 监听剪贴板，自动将多张图片合成为GIF动画。
// 支持BMP/PNG/JPG格式，包括4位BMP等特殊格式。
package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	// 图片格式支持
	_ "image/jpeg"
	_ "image/png"
	_ "github.com/jsummers/gobmp" // 支持1/4/8/16/24/32位BMP及RLE压缩

	// 第三方库
	"github.com/nfnt/resize"
	"golang.org/x/sys/windows"
)

const (
	pollInterval  = 300    // 剪贴板轮询间隔(ms)
	gifDelay      = 50     // GIF帧延迟(10ms为单位, 50=500ms)
	minImages     = 2      // 最少图片数
	maxRetries    = 3      // 剪贴板访问重试次数
	margin        = 10     // 帧间距(px)
	tempDir       = "temp" // 临时文件目录
	maxPathLength = 32767  // Windows路径长度上限
	cfHDrop       = 15     // Windows剪贴板文件格式常量
)

var supportedImageFormats = map[string]bool{
	".bmp":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	getClipboardData = user32.NewProc("GetClipboardData")
	shell32          = windows.NewLazySystemDLL("shell32.dll")
	dragQueryFileW   = shell32.NewProc("DragQueryFileW")
)

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   ClipTool - 剪贴板图片转GIF工具       ║")
	fmt.Println("║   支持BMP/PNG/JPG，按Ctrl+C退出        ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	clearTempDirectory()
	defer clearTempDirectory()

	fmt.Println("✓ 运行中，监听剪贴板...")
	monitorClipboard()
}

func monitorClipboard() {
	var lastClipboardHash string
	ticker := time.NewTicker(time.Millisecond * pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		imageFiles, currentHash := getClipboardFiles()

		if currentHash == lastClipboardHash || len(imageFiles) < minImages {
			continue
		}
		lastClipboardHash = currentHash

		fmt.Printf("\n► 检测到 %d 张图片，生成GIF中...\n", len(imageFiles))
		startTime := time.Now()

		if err := processAndSave(imageFiles); err != nil {
			fmt.Printf("✗ 错误: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("✓ GIF已生成，可以粘贴了！(耗时: %.2fs)\n", elapsed.Seconds())
	}
}

func getClipboardFiles() (imageFiles []string, contentHash string) {
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
		dragQueryFileW.Call(hdropHandle, uintptr(fileIndex), uintptr(unsafe.Pointer(&pathBuffer[0])), uintptr(len(pathBuffer)))

		filePath := windows.UTF16ToString(pathBuffer)
		fileExt := strings.ToLower(filepath.Ext(filePath))
		if supportedImageFormats[fileExt] {
			imageFiles = append(imageFiles, filePath)
		}
	}

	contentHash = calcHash(imageFiles)
	return imageFiles, contentHash
}

func clearTempDirectory() {
	if _, err := os.Stat(tempDir); err == nil {
		os.RemoveAll(tempDir)
	}
	os.MkdirAll(tempDir, 0755)
}

func setClipboardGIF(gifData []byte) error {
	timestamp := time.Now().Format("20060102_150405")
	tempFilePath := filepath.Join(tempDir, fmt.Sprintf("temp_clipboard_%s.gif", timestamp))

	absPath, err := filepath.Abs(tempFilePath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	if err := os.WriteFile(tempFilePath, gifData, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	cmd := exec.Command("powershell.exe", "-Command", fmt.Sprintf("Set-Clipboard -Path '%s'", absPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PowerShell设置剪贴板失败: %v, 输出: %s", err, string(output))
	}

	return nil
}

func processAndSave(files []string) error {
	images, err := loadImages(files)
	if err != nil {
		return err
	}

	if len(images) < minImages {
		return fmt.Errorf("有效图片不足 %d 张（实际: %d）", minImages, len(images))
	}

	frames := createFrames(images)
	gifData, err := encodeGIF(frames)
	if err != nil {
		return err
	}

	return setClipboardGIF(gifData)
}

func loadImages(files []string) ([]image.Image, error) {
	var images []image.Image

	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("  ⚠ 跳过: %s (%v)\n", filepath.Base(filePath), err)
			continue
		}

		img, format, err := image.Decode(file)
		file.Close()

		if err != nil {
			fmt.Printf("  ⚠ 跳过: %s (解码失败)\n", filepath.Base(filePath))
			continue
		}

		fmt.Printf("  ✓ 已加载: %s (%s)\n", filepath.Base(filePath), format)
		images = append(images, img)
	}

	return images, nil
}

// 布局: [所有缩略图] + [当前帧]
func createFrames(sourceImages []image.Image) []image.Image {
	if len(sourceImages) == 0 {
		return nil
	}

	firstImageBounds := sourceImages[0].Bounds()
	standardWidth := firstImageBounds.Dx()
	standardHeight := firstImageBounds.Dy()

	resizedImages := make([]image.Image, len(sourceImages))
	for i, img := range sourceImages {
		resizedImages[i] = resize.Resize(uint(standardWidth), uint(standardHeight), img, resize.Bilinear)
	}

	totalColumns := len(sourceImages) + 1
	canvasWidth := standardWidth*totalColumns + margin*(totalColumns-1)
	canvasHeight := standardHeight

	frames := make([]image.Image, len(sourceImages))
	for frameIndex := 0; frameIndex < len(sourceImages); frameIndex++ {
		canvas := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

		for thumbIndex := 0; thumbIndex < len(resizedImages); thumbIndex++ {
			xPosition := thumbIndex * (standardWidth + margin)
			targetRect := image.Rect(xPosition, 0, xPosition+standardWidth, standardHeight)
			draw.Draw(canvas, targetRect, resizedImages[thumbIndex], image.Point{}, draw.Src)
		}

		rightXPosition := (totalColumns - 1) * (standardWidth + margin)
		rightTargetRect := image.Rect(rightXPosition, 0, rightXPosition+standardWidth, standardHeight)
		draw.Draw(canvas, rightTargetRect, resizedImages[frameIndex], image.Point{}, draw.Src)

		frames[frameIndex] = canvas
	}

	return frames
}

func encodeGIF(frames []image.Image) ([]byte, error) {
	if frames == nil || len(frames) == 0 {
		return nil, fmt.Errorf("没有帧可供编码")
	}

	animation := &gif.GIF{
		Image: make([]*image.Paletted, len(frames)),
		Delay: make([]int, len(frames)),
	}

	palette := generatePalette()

	for frameIndex, frame := range frames {
		frameBounds := frame.Bounds()
		palettedFrame := image.NewPaletted(frameBounds, palette)
		draw.Src.Draw(palettedFrame, frameBounds, frame, image.Point{})

		animation.Image[frameIndex] = palettedFrame
		animation.Delay[frameIndex] = gifDelay
	}

	animation.LoopCount = 0

	buffer := new(bytes.Buffer)
	if err := gif.EncodeAll(buffer, animation); err != nil {
		return nil, fmt.Errorf("GIF编码失败: %v", err)
	}

	return buffer.Bytes(), nil
}

func generatePalette() color.Palette {
	const (
		rgbLevels   = 6
		rgbStep     = 51
		grayColors  = 40
		totalColors = 256
	)

	palette := make(color.Palette, totalColors)
	colorIndex := 0

	for r := 0; r < rgbLevels; r++ {
		for g := 0; g < rgbLevels; g++ {
			for b := 0; b < rgbLevels; b++ {
				palette[colorIndex] = color.RGBA{
					R: uint8(r * rgbStep),
					G: uint8(g * rgbStep),
					B: uint8(b * rgbStep),
					A: 255,
				}
				colorIndex++
			}
		}
	}

	for i := 0; i < grayColors; i++ {
		grayValue := uint8(i * 255 / (grayColors - 1))
		palette[colorIndex] = color.RGBA{
			R: grayValue,
			G: grayValue,
			B: grayValue,
			A: 255,
		}
		colorIndex++
	}

	return palette
}

func calcHash(files []string) string {
	if len(files) == 0 {
		return ""
	}

	hasher := md5.New()
	for _, filePath := range files {
		hasher.Write([]byte(filePath))
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}
