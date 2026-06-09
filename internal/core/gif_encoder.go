package core

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"

	"cliptool/internal/applog"

	"github.com/nfnt/resize"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	thumbMargin     = 1
	separatorMargin = 10
)

func EncodeFiles(files []string, options GifOptions) ([]byte, error) {
	applog.Debugf("EncodeFiles 开始: files=%d delay=%d", len(files), options.Normalized().DelayMS)
	loadedImages, err := LoadImages(files)
	if err != nil {
		applog.Errorf("EncodeFiles 读取图片失败: files=%d err=%v", len(files), err)
		return nil, err
	}

	images := make([]image.Image, 0, len(loadedImages))
	for _, loaded := range loadedImages {
		images = append(images, loaded.Image)
	}
	applog.Debugf("EncodeFiles 图片读取完成: loaded=%d", len(images))
	return EncodeImages(images, options)
}

func EncodeImages(sourceImages []image.Image, options GifOptions) ([]byte, error) {
	if len(sourceImages) < MinImages {
		applog.Warnf("EncodeImages 有效图片不足: images=%d min=%d", len(sourceImages), MinImages)
		return nil, fmt.Errorf("有效图片不足 %d 张（实际: %d）", MinImages, len(sourceImages))
	}

	frames := createFrames(sourceImages)
	applog.Debugf("GIF 帧创建完成: sourceImages=%d frames=%d", len(sourceImages), len(frames))
	return encodeGIF(frames, options)
}

func createFrames(sourceImages []image.Image) []image.Image {
	firstImageBounds := sourceImages[0].Bounds()
	standardWidth := firstImageBounds.Dx()
	standardHeight := firstImageBounds.Dy()

	resizedImages := make([]image.Image, len(sourceImages))
	for i, img := range sourceImages {
		resizedImages[i] = resize.Resize(uint(standardWidth), uint(standardHeight), img, resize.Bilinear)
	}

	thumbCount := len(sourceImages)
	canvasWidth := thumbCount*standardWidth + (thumbCount-1)*thumbMargin + separatorMargin + standardWidth
	canvasHeight := standardHeight

	frames := make([]image.Image, len(sourceImages))
	for frameIndex := range sourceImages {
		canvas := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

		for thumbIndex := range resizedImages {
			xPosition := thumbIndex * (standardWidth + thumbMargin)
			targetRect := image.Rect(xPosition, 0, xPosition+standardWidth, standardHeight)
			draw.Draw(canvas, targetRect, resizedImages[thumbIndex], image.Point{}, draw.Src)
		}

		rightXPosition := thumbCount*standardWidth + (thumbCount-1)*thumbMargin + separatorMargin
		rightTargetRect := image.Rect(rightXPosition, 0, rightXPosition+standardWidth, standardHeight)
		draw.Draw(canvas, rightTargetRect, resizedImages[frameIndex], image.Point{}, draw.Src)
		drawFrameNumber(canvas, rightXPosition, standardWidth, frameIndex+1)

		frames[frameIndex] = canvas
	}

	return frames
}

func drawFrameNumber(canvas *image.RGBA, rightXPosition, standardWidth, frameNumber int) {
	textColor := image.NewUniform(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	label := fmt.Sprintf("%d", frameNumber)

	for dx := 0; dx <= 1; dx++ {
		for dy := 0; dy <= 1; dy++ {
			drawer := &font.Drawer{
				Dst:  canvas,
				Src:  textColor,
				Face: basicfont.Face7x13,
				Dot:  fixed.P(rightXPosition+standardWidth-25+dx, 15+dy),
			}
			drawer.DrawString(label)
		}
	}
}

func encodeGIF(frames []image.Image, options GifOptions) ([]byte, error) {
	if len(frames) == 0 {
		applog.Warnf("GIF 编码失败: 没有帧")
		return nil, fmt.Errorf("没有帧可供编码")
	}

	animation := &gif.GIF{
		Image: make([]*image.Paletted, len(frames)),
		Delay: make([]int, len(frames)),
	}

	palette := generatePalette()
	delayUnits := options.DelayUnits()
	applog.Debugf("GIF 编码开始: frames=%d delayUnits=%d delayMS=%d", len(frames), delayUnits, options.Normalized().DelayMS)

	for frameIndex, frame := range frames {
		frameBounds := frame.Bounds()
		palettedFrame := image.NewPaletted(frameBounds, palette)
		draw.Src.Draw(palettedFrame, frameBounds, frame, image.Point{})

		animation.Image[frameIndex] = palettedFrame
		animation.Delay[frameIndex] = delayUnits
	}

	animation.LoopCount = 0

	buffer := new(bytes.Buffer)
	if err := gif.EncodeAll(buffer, animation); err != nil {
		applog.Errorf("GIF 编码失败: frames=%d err=%v", len(frames), err)
		return nil, fmt.Errorf("GIF 编码失败: %v", err)
	}

	applog.Debugf("GIF 编码完成: frames=%d bytes=%d", len(frames), buffer.Len())
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
