package core

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"cliptool/internal/applog"

	_ "image/jpeg"
	_ "image/png"

	_ "github.com/jsummers/gobmp"
)

type LoadedImage struct {
	Image  image.Image
	Path   string
	Name   string
	Format string
	Width  int
	Height int
}

type rawDataProfile struct {
	name          string
	width, height int
	dtype         string
	bits          int
}

var standardImageExts = map[string]bool{
	".bmp":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

var rawDataProfiles = map[int64]rawDataProfile{
	43808:  {"RAW", 148, 148, "uint16", 12},
	102400: {"BIN", 160, 160, "uint32", 16},
}

func IsSupportedImage(filePath string) bool {
	return isStandardImage(filePath) || isRawDataImage(filePath)
}

func LoadImage(filePath string) (image.Image, string, error) {
	if isRawDataImage(filePath) {
		applog.Debugf("按原始指纹数据读取图片: path=%q", filePath)
		return loadRawDataImage(filePath)
	}
	applog.Debugf("按标准图片格式读取图片: path=%q ext=%q", filePath, filepath.Ext(filePath))
	return loadStandardImage(filePath)
}

func LoadImages(files []string) ([]LoadedImage, error) {
	images := make([]LoadedImage, 0, len(files))
	for _, filePath := range files {
		img, format, err := LoadImage(filePath)
		if err != nil {
			applog.Warnf("跳过无法读取的图片: path=%q err=%v", filePath, err)
			continue
		}

		bounds := img.Bounds()
		images = append(images, LoadedImage{
			Image:  img,
			Path:   filePath,
			Name:   filepath.Base(filePath),
			Format: format,
			Width:  bounds.Dx(),
			Height: bounds.Dy(),
		})
		applog.Debugf("图片读取成功: path=%q format=%q width=%d height=%d", filePath, format, bounds.Dx(), bounds.Dy())
	}

	if len(images) < MinImages {
		applog.Warnf("有效图片不足: input=%d loaded=%d min=%d", len(files), len(images), MinImages)
		return images, fmt.Errorf("有效图片不足 %d 张（实际: %d）", MinImages, len(images))
	}
	return images, nil
}

func isStandardImage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return standardImageExts[ext]
}

func loadStandardImage(filePath string) (image.Image, string, error) {
	info, statErr := os.Stat(filePath)
	if statErr != nil {
		applog.Warnf("标准图片文件状态检查失败: path=%q err=%v", filePath, statErr)
	} else {
		applog.Debugf("标准图片文件状态: path=%q size=%d modified=%s", filePath, info.Size(), info.ModTime().Format("2006-01-02T15:04:05.000Z07:00"))
	}

	file, err := os.Open(filePath)
	if err != nil {
		applog.Errorf("打开标准图片失败: path=%q err=%v", filePath, err)
		return nil, "", err
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		applog.Errorf("解码标准图片失败: path=%q err=%v", filePath, err)
		return nil, "", err
	}
	bounds := img.Bounds()
	applog.Debugf("解码标准图片成功: path=%q format=%q width=%d height=%d", filePath, format, bounds.Dx(), bounds.Dy())
	return img, format, nil
}

func isRawDataImage(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	_, ok := rawDataProfiles[info.Size()]
	return ok
}

func loadRawDataImage(filePath string) (image.Image, string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		applog.Errorf("原始指纹数据文件状态检查失败: path=%q err=%v", filePath, err)
		return nil, "", err
	}
	applog.Debugf("原始指纹数据文件状态: path=%q size=%d modified=%s", filePath, info.Size(), info.ModTime().Format("2006-01-02T15:04:05.000Z07:00"))

	profile, ok := rawDataProfiles[info.Size()]
	if !ok {
		applog.Warnf("未知原始指纹数据尺寸: path=%q size=%d", filePath, info.Size())
		return nil, "", fmt.Errorf("unknown raw data size: %d", info.Size())
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		applog.Errorf("读取原始指纹数据失败: path=%q err=%v", filePath, err)
		return nil, "", err
	}

	img := image.NewGray(image.Rect(0, 0, profile.width, profile.height))
	pixelCount := profile.width * profile.height

	for i := 0; i < pixelCount; i++ {
		var rawValue uint32
		if profile.dtype == "uint16" {
			rawValue = uint32(binary.LittleEndian.Uint16(data[i*2:]))
		} else {
			rawValue = binary.LittleEndian.Uint32(data[i*4:])
		}
		img.Pix[i] = uint8(rawValue >> (profile.bits - 8))
	}

	format := fmt.Sprintf("%s/%dx%d", profile.name, profile.width, profile.height)
	applog.Debugf("原始指纹数据读取成功: path=%q format=%q bytes=%d", filePath, format, len(data))
	return img, format, nil
}
