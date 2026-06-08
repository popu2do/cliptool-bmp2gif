package main

import (
	"embed"
	"log"

	"cliptool/internal/clipboard"
	"cliptool/internal/session"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp(session.NewFrameStore(), clipboard.NewService("temp"))

	err := wails.Run(&options.App{
		Title:     "ClipTool 多图合成 GIF",
		Width:     680,
		Height:    460,
		MinWidth:  560,
		MinHeight: 380,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 246, G: 247, B: 249, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
