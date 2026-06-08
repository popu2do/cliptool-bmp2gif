package core

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

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
		return loadRawDataImage(filePath)
	}
	return loadStandardImage(filePath)
}

func LoadImages(files []string) ([]LoadedImage, error) {
	images := make([]LoadedImage, 0, len(files))
	for _, filePath := range files {
		img, format, err := LoadImage(filePath)
		if err != nil {
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
	}

	if len(images) < MinImages {
		return images, fmt.Errorf("有效图片不足 %d 张（实际: %d）", MinImages, len(images))
	}
	return images, nil
}

func isStandardImage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return standardImageExts[ext]
}

func loadStandardImage(filePath string) (image.Image, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	return image.Decode(file)
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
