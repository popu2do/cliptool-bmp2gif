package core

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

func TestEncodeImagesRequiresAtLeastTwoImages(t *testing.T) {
	_, err := EncodeImages([]image.Image{solidImage(color.White)}, GifOptions{})
	if err == nil {
		t.Fatal("EncodeImages() error = nil, want error")
	}
}

func TestEncodeImagesUsesInputOrderAndDelay(t *testing.T) {
	data, err := EncodeImages([]image.Image{
		solidImage(color.RGBA{R: 255, A: 255}),
		solidImage(color.RGBA{G: 255, A: 255}),
	}, GifOptions{DelayMS: 700})
	if err != nil {
		t.Fatalf("EncodeImages() error = %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeAll() error = %v", err)
	}

	if len(decoded.Image) != 2 {
		t.Fatalf("frame count = %d, want 2", len(decoded.Image))
	}
	for i, delay := range decoded.Delay {
		if delay != 70 {
			t.Fatalf("delay[%d] = %d, want 70", i, delay)
		}
	}
}

func solidImage(c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}
