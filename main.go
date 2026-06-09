package main

import (
	"embed"
	"os"
	"runtime"

	"cliptool/internal/applog"
	"cliptool/internal/clipboard"
	"cliptool/internal/session"

	"github.com/wailsapp/wails/v2"
	wailslogger "github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logPath, err := applog.Init("cliptool")
	if err != nil {
		os.Exit(1)
	}
	defer applog.Close()

	workingDir, _ := os.Getwd()
	executablePath, _ := os.Executable()
	applog.Infof("程序启动: exe=%q cwd=%q os=%s arch=%s args=%q log=%q", executablePath, workingDir, runtime.GOOS, runtime.GOARCH, os.Args, logPath)

	app := NewApp(session.NewFrameStore(), clipboard.NewService("temp"))

	err = wails.Run(&options.App{
		Title:     "ClipTool 多图合成 GIF",
		Width:     680,
		Height:    460,
		MinWidth:  560,
		MinHeight: 380,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:   &options.RGBA{R: 246, G: 247, B: 249, A: 1},
		Logger:             applog.NewWailsLogger(),
		LogLevel:           wailslogger.DEBUG,
		LogLevelProduction: wailslogger.DEBUG,
		OnStartup:          app.startup,
		OnShutdown:         app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		applog.Errorf("程序异常退出: %v", err)
		os.Exit(1)
	}
}
