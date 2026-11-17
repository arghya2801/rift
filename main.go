package main

import (
	"embed"
	"log"
	"path/filepath"

	"rift/internal/store"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:dist
var assets embed.FS

func main() {
	if runHelper() {
		return
	}
	app := NewApp(store.ConfigDir())
	err := wails.Run(&options.App{
		Title:            "Rift",
		Width:            1280,
		Height:           800,
		MinWidth:         640,
		MinHeight:        400,
		BackgroundColour: &options.RGBA{R: 11, G: 13, B: 16, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		Bind:             []any{app},
		// GPU process costs ~110 MB private memory; a flat UI renders fine in software.
		Windows: &windows.Options{Theme: windows.Dark, WebviewUserDataPath: filepath.Join(store.ConfigDir(), "webview"), WebviewGpuIsDisabled: true},
	})
	if err != nil {
		log.Fatal(err)
	}
}
