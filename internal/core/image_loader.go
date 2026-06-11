package core

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

type ImageLoader interface {
	Name() string
	IsSupported(filePath string) bool
	Load(filePath string) (image.Image, string, error)
}

var (
	imageLoadersMu sync.RWMutex
	imageLoaders   []ImageLoader
)

var standardImageExts = map[string]bool{
	".bmp":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

func RegisterImageLoader(loader ImageLoader) {
	imageLoadersMu.Lock()
	defer imageLoadersMu.Unlock()
	imageLoaders = append(imageLoaders, loader)
	applog.Debugf("注册图片扩展解析器: name=%q", loader.Name())
}

func IsSupportedImage(filePath string) bool {
	return isStandardImage(filePath) || findExtensionLoader(filePath) != nil
}

func LoadImage(filePath string) (image.Image, string, error) {
	if isStandardImage(filePath) {
		applog.Debugf("按标准图片格式读取图片: path=%q ext=%q", filePath, filepath.Ext(filePath))
		return loadStandardImage(filePath)
	}

	if loader := findExtensionLoader(filePath); loader != nil {
		applog.Debugf("按扩展解析器读取图片: loader=%q path=%q", loader.Name(), filePath)
		return loader.Load(filePath)
	}

	return nil, "", fmt.Errorf("不支持的图片格式或尺寸: %s", filepath.Base(filePath))
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

func findExtensionLoader(filePath string) ImageLoader {
	imageLoadersMu.RLock()
	defer imageLoadersMu.RUnlock()
	for _, loader := range imageLoaders {
		if loader.IsSupported(filePath) {
			return loader
		}
	}
	return nil
}
