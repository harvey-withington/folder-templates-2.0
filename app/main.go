// Command FolderTemplates is the Folder Templates desktop app.
package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/harvey-withington/folder-templates-2.0/app/internal/launch"
	"github.com/harvey-withington/folder-templates-2.0/app/internal/settings"
	"github.com/harvey-withington/folder-templates-2.0/app/internal/shell"
)

//go:embed all:frontend/dist
var assets embed.FS

// Version is set at build time with -ldflags "-X main.Version=…".
var Version = "dev"

func main() {
	// Installer hooks: set up or remove Explorer integration and exit.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--shell-install":
			os.Exit(shellInstall(os.Args[2:]))
		case "--shell-uninstall":
			os.Exit(shellUninstall())
		case "--serve-ui":
			// Serve the UI to a browser instead of opening a window
			// (scripts/screenshots.mjs uses this with headless Edge).
			if len(os.Args) < 3 {
				log.Fatal("usage: FolderTemplates --serve-ui <host:port> [launch args]")
			}
			log.Fatal(serveUI(os.Args[2], os.Args[3:]))
		}
	}

	intent := launch.Parse(os.Args[1:])

	path, err := settings.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	store, err := settings.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	width, height := 1040, 700
	startHidden := false
	if w := store.Get().Window; w != nil {
		width, height = w.Width, w.Height
		startHidden = true // shown in domReady once positioned
	}

	app := NewApp(intent, store, findSamples())
	err = wails.Run(&options.App{
		Title:            "Folder Templates",
		Width:            width,
		Height:           height,
		MinWidth:         720,
		MinHeight:        480,
		StartHidden:      startHidden,
		BackgroundColour: &options.RGBA{R: 24, G: 24, B: 27, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnBeforeClose:    app.beforeClose,
		DragAndDrop:      &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true},
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// shellInstall enables the named features (comma-separated, or "all").
func shellInstall(args []string) int {
	in, err := shell.Default()
	if err != nil {
		log.Print(err)
		return 1
	}
	want := shell.Features
	if len(args) > 0 && args[0] != "all" {
		want = nil
		for _, f := range strings.Split(args[0], ",") {
			want = append(want, shell.Feature(strings.TrimSpace(f)))
		}
	}
	code := 0
	for _, f := range want {
		if err := in.Set(f, true); err != nil {
			log.Printf("%s: %v", f, err)
			code = 1
		}
	}
	return code
}

// shellUninstall removes every Explorer integration.
func shellUninstall() int {
	in, err := shell.Default()
	if err == nil {
		err = in.RemoveAll()
	}
	if err != nil {
		log.Print(err)
		return 1
	}
	return 0
}

// findSamples locates the bundled sample templates: "samples" beside the
// executable when installed, or the repo's samples folder during development
// (the dev binary lives in app/build/bin).
func findSamples() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	for i := 0; i < 4; i++ {
		candidate := filepath.Join(dir, "samples")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
