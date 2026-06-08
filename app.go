package main

import (
	"context"
	"fmt"

	"cliptool/internal/clipboard"
	"cliptool/internal/core"
	"cliptool/internal/session"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx               context.Context
	store             *session.FrameStore
	clipboard         *clipboard.Service
	lastClipboardHash string
}

type GenerateResult struct {
	OK         bool
	Message    string
	FrameCount int
}

func NewApp(store *session.FrameStore, clipboardService *clipboard.Service) *App {
	return &App{
		store:     store,
		clipboard: clipboardService,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.clipboard.ClearTempDirectory()
}

func (a *App) shutdown(ctx context.Context) {
	a.clipboard.ClearTempDirectory()
}

func (a *App) GetFrames() []session.FrameItem {
	return a.store.Frames()
}

func (a *App) ScanClipboard() session.AddResult {
	paths, contentHash := a.clipboard.ReadImageFiles()
	if contentHash == "" || contentHash == a.lastClipboardHash {
		return session.AddResult{
			Frames:  a.store.Frames(),
			Message: "监听中",
		}
	}

	a.lastClipboardHash = contentHash
	return a.store.AddPaths(paths)
}

func (a *App) RemoveFrame(id string) []session.FrameItem {
	return a.store.Remove(id)
}

func (a *App) ReorderFrames(ids []string) []session.FrameItem {
	return a.store.Reorder(ids)
}

func (a *App) ClearFrames() {
	a.markCurrentClipboardAsSeen()
	a.store.Clear()
}

func (a *App) GenerateGIF(options core.GifOptions) GenerateResult {
	paths := a.store.Paths()
	if len(paths) < core.MinImages {
		return GenerateResult{
			OK:         false,
			Message:    fmt.Sprintf("至少需要 %d 张图片", core.MinImages),
			FrameCount: len(paths),
		}
	}

	gifData, err := core.EncodeFiles(paths, options)
	if err != nil {
		return GenerateResult{
			OK:         false,
			Message:    err.Error(),
			FrameCount: len(paths),
		}
	}

	if err := a.clipboard.WriteGIF(gifData); err != nil {
		return GenerateResult{
			OK:         false,
			Message:    err.Error(),
			FrameCount: len(paths),
		}
	}

	frameCount := len(paths)
	currentClipboardHash := a.lastClipboardHash
	a.store.Clear()
	a.lastClipboardHash = currentClipboardHash
	return GenerateResult{
		OK:         true,
		Message:    "GIF 已复制到剪贴板",
		FrameCount: frameCount,
	}
}

func (a *App) markCurrentClipboardAsSeen() {
	_, contentHash := a.clipboard.ReadImageFiles()
	a.lastClipboardHash = contentHash
}

func (a *App) SetAlwaysOnTop(enabled bool) {
	if a.ctx != nil {
		runtime.WindowSetAlwaysOnTop(a.ctx, enabled)
	}
}
