package main

import (
	"context"
	"fmt"

	"cliptool/internal/applog"
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
	applog.Infof("Wails 启动完成，清理临时目录")
	a.clipboard.ClearTempDirectory()
}

func (a *App) shutdown(ctx context.Context) {
	applog.Infof("程序关闭，清理临时目录")
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

	applog.Debugf("发现新的剪切板图片批次: paths=%d contentHash=%q lastHash=%q", len(paths), contentHash, a.lastClipboardHash)
	a.lastClipboardHash = contentHash
	result := a.store.AddPaths(paths)
	applog.Infof("追加剪切板图片完成: added=%d skipped=%d total=%d message=%q", result.Added, result.Skipped, len(result.Frames), result.Message)
	return result
}

func (a *App) RemoveFrame(id string) []session.FrameItem {
	return a.store.Remove(id)
}

func (a *App) ReorderFrames(ids []string) []session.FrameItem {
	return a.store.Reorder(ids)
}

func (a *App) ClearFrames() {
	applog.Infof("用户清空帧列表")
	a.markCurrentClipboardAsSeen()
	a.store.Clear()
}

func (a *App) GenerateGIF(options core.GifOptions) GenerateResult {
	paths := a.store.Paths()
	applog.Infof("开始生成 GIF: frameCount=%d delay=%d", len(paths), options.DelayMS)
	if len(paths) < core.MinImages {
		applog.Warnf("生成 GIF 失败: 有效图片不足 frameCount=%d min=%d", len(paths), core.MinImages)
		return GenerateResult{
			OK:         false,
			Message:    fmt.Sprintf("至少需要 %d 张图片", core.MinImages),
			FrameCount: len(paths),
		}
	}

	gifData, err := core.EncodeFiles(paths, options)
	if err != nil {
		applog.Errorf("生成 GIF 编码失败: %v", err)
		return GenerateResult{
			OK:         false,
			Message:    err.Error(),
			FrameCount: len(paths),
		}
	}

	if err := a.clipboard.WriteGIF(gifData); err != nil {
		applog.Errorf("写入 GIF 到剪切板失败: %v", err)
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
	applog.Infof("生成 GIF 成功: frameCount=%d gifBytes=%d", frameCount, len(gifData))
	return GenerateResult{
		OK:         true,
		Message:    "GIF 已复制到剪贴板",
		FrameCount: frameCount,
	}
}

func (a *App) markCurrentClipboardAsSeen() {
	_, contentHash := a.clipboard.ReadImageFiles()
	a.lastClipboardHash = contentHash
	applog.Debugf("标记当前剪切板为已处理: contentHash=%q", contentHash)
}

func (a *App) SetAlwaysOnTop(enabled bool) {
	if a.ctx != nil {
		runtime.WindowSetAlwaysOnTop(a.ctx, enabled)
	}
}
