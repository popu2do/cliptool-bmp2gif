package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"cliptool/internal/applog"

	"github.com/nfnt/resize"
)

const thumbnailMaxSize = 140

func ThumbnailDataURL(img image.Image) (string, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var thumb image.Image
	if width >= height {
		thumb = resize.Resize(thumbnailMaxSize, 0, img, resize.Bilinear)
	} else {
		thumb = resize.Resize(0, thumbnailMaxSize, img, resize.Bilinear)
	}

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, thumb); err != nil {
		applog.Errorf("生成缩略图失败: sourceWidth=%d sourceHeight=%d err=%v", width, height, err)
		return "", fmt.Errorf("生成缩略图失败: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(buffer.Bytes())
	applog.Debugf("生成缩略图成功: sourceWidth=%d sourceHeight=%d bytes=%d", width, height, buffer.Len())
	return "data:image/png;base64," + encoded, nil
}
