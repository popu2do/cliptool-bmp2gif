package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	_ "github.com/jsummers/gobmp"
)

// ============================================================
// 标准图片格式 - 按后缀识别
// ============================================================

var standardImageExts = map[string]bool{
	".bmp":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

func isStandardImage(filePath string) bool {
	ext := strings.ToLower(filePath)
	for e := range standardImageExts {
		if strings.HasSuffix(ext, e) {
			return true
		}
	}
	return false
}

func loadStandardImage(filePath string) (image.Image, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	return image.Decode(file)
}

// ============================================================
// 原始数据图 - 按文件大小识别 (RAW/BIN 指纹图)
// ============================================================

type rawDataProfile struct {
	name          string
	width, height int
	dtype         string // "uint16" or "uint32"
	bits          int
}

var rawDataProfiles = map[int64]rawDataProfile{
	43808:  {"RAW", 148, 148, "uint16", 12},
	102400: {"BIN", 160, 160, "uint32", 16},
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
		return nil, "", err
	}

	profile, ok := rawDataProfiles[info.Size()]
	if !ok {
		return nil, "", fmt.Errorf("unknown raw data size: %d", info.Size())
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
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
	return img, format, nil
}

// ============================================================
// 统一入口
// ============================================================

func IsSupportedImage(filePath string) bool {
	return isStandardImage(filePath) || isRawDataImage(filePath)
}

func LoadImage(filePath string) (image.Image, string, error) {
	if isRawDataImage(filePath) {
		return loadRawDataImage(filePath)
	}
	return loadStandardImage(filePath)
}
