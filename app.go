package main

import (
	"context"
	"fmt"
	"path/filepath"

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

func (a *App) AddDroppedFiles(paths []string) session.AddResult {
	applog.Infof("拖拽导入文件: count=%d", len(paths))
	result := a.store.AddPaths(paths)
	if result.Error {
		applog.Warnf("拖拽导入部分失败: added=%d skipped=%d message=%q", result.Added, result.Skipped, result.Message)
	} else {
		applog.Infof("拖拽导入完成: added=%d skipped=%d total=%d message=%q", result.Added, result.Skipped, len(result.Frames), result.Message)
	}
	return result
}

func (a *App) ScanClipboard() session.AddResult {
	clipboardResult := a.clipboard.ReadImageFiles()
	if clipboardResult.ContentHash == "" || clipboardResult.ContentHash == a.lastClipboardHash {
		return session.AddResult{
			Frames:  a.store.Frames(),
			Message: "监听中",
		}
	}

	applog.Debugf("发现新的剪切板图片批次: paths=%d unsupported=%d contentHash=%q lastHash=%q", len(clipboardResult.ImageFiles), len(clipboardResult.UnsupportedFiles), clipboardResult.ContentHash, a.lastClipboardHash)
	a.lastClipboardHash = clipboardResult.ContentHash
	if clipboardResult.ClipboardOpenFailed {
		return session.AddResult{
			Frames:  a.store.Frames(),
			Message: "剪贴板正被其他程序占用，稍后会继续监听",
			Error:   true,
		}
	}
	if clipboardResult.FileListUnreadable {
		return session.AddResult{
			Frames:  a.store.Frames(),
			Message: "剪贴板里有文件列表，但当前无法读取；请重新复制文件，或先用拖拽添加",
			Error:   true,
		}
	}
	if clipboardResult.NonFileContent {
		return session.AddResult{
			Frames:  a.store.Frames(),
			Message: "剪贴板里没有文件列表，请在资源管理器里选中图片文件后复制",
			Error:   true,
		}
	}
	if len(clipboardResult.ImageFiles) == 0 {
		if len(clipboardResult.UnsupportedFiles) == 0 {
			return session.AddResult{
				Frames:  a.store.Frames(),
				Message: "监听中",
			}
		}
		return session.AddResult{
			Frames:      a.store.Frames(),
			Unsupported: len(clipboardResult.UnsupportedFiles),
			Message:     unsupportedMessage(clipboardResult.UnsupportedFiles),
			Error:       len(clipboardResult.UnsupportedFiles) > 0,
		}
	}

	result := a.store.AddPaths(clipboardResult.ImageFiles)
	result.Unsupported = len(clipboardResult.UnsupportedFiles)
	if result.Unsupported > 0 {
		result.Message = fmt.Sprintf("%s；%d 项无法解析", result.Message, result.Unsupported)
		result.Error = true
	}
	applog.Infof("追加剪切板图片完成: added=%d skipped=%d unsupported=%d total=%d message=%q", result.Added, result.Skipped, result.Unsupported, len(result.Frames), result.Message)
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
	result := a.clipboard.ReadImageFiles()
	a.lastClipboardHash = result.ContentHash
	applog.Debugf("标记当前剪切板为已处理: contentHash=%q", result.ContentHash)
}

func unsupportedMessage(files []clipboard.UnsupportedFile) string {
	if len(files) == 0 {
		return "未发现可用图片"
	}

	first := files[0]
	name := first.Name
	if name == "" {
		name = filepath.Base(first.Path)
	}
	if first.IsDir {
		if len(files) == 1 {
			return fmt.Sprintf("复制到的是文件夹：%s，请进入文件夹后选中图片文件复制", name)
		}
		return fmt.Sprintf("无法解析 %d 项，包含文件夹 %s；请进入文件夹后选中图片文件复制", len(files), name)
	}
	if len(files) == 1 {
		return fmt.Sprintf("无法解析：%s（%d 字节）", name, first.Size)
	}
	return fmt.Sprintf("无法解析 %d 个文件，例如 %s（%d 字节）", len(files), name, first.Size)
}

func (a *App) SetAlwaysOnTop(enabled bool) {
	if a.ctx != nil {
		runtime.WindowSetAlwaysOnTop(a.ctx, enabled)
	}
}
