package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	ft "github.com/harvey-withington/folder-templates-2.0/engine"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/harvey-withington/folder-templates-2.0/app/internal/launch"
	"github.com/harvey-withington/folder-templates-2.0/app/internal/library"
	"github.com/harvey-withington/folder-templates-2.0/app/internal/placement"
	"github.com/harvey-withington/folder-templates-2.0/app/internal/settings"
	"github.com/harvey-withington/folder-templates-2.0/app/internal/shell"
)

// App is the service bound to the frontend. Every exported method becomes a
// window.go.main.App.<Method> call returning a promise.
type App struct {
	ctx      context.Context
	intent   launch.Intent
	settings *settings.Store
	samples  string
	monitor  placement.Monitor

	mu        sync.Mutex
	genCancel context.CancelFunc

	// headless: served to a browser by --serve-ui, with no Wails window,
	// so calls into the Wails runtime are skipped.
	headless bool
}

func NewApp(intent launch.Intent, store *settings.Store, samplesDir string, monitor placement.Monitor) *App {
	return &App{intent: intent, settings: store, samples: samplesDir, monitor: monitor}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Each extra folder from a multi-selection SendTo gets its own window.
	if exe, err := os.Executable(); err == nil {
		for _, extra := range a.intent.Extra {
			_ = exec.Command(exe, launch.ArgsFor(a.intent, extra)...).Start()
		}
	}
}

// domReady centres the (still hidden) window on the display the app was
// launched from, at the remembered size, then shows it.
func (a *App) domReady(ctx context.Context) {
	width, height, maximised := defaultWidth, defaultHeight, false
	if w := a.settings.Get().Window; w != nil {
		width, height, maximised = w.Width, w.Height, w.Maximised
	}
	if err := placement.Place(a.monitor, width, height); err != nil {
		runtime.WindowCenter(ctx)
	}
	if maximised {
		runtime.WindowMaximise(ctx)
	}
	runtime.WindowShow(ctx)
}

func (a *App) beforeClose(ctx context.Context) bool {
	maximised := runtime.WindowIsMaximised(ctx)
	w, h := runtime.WindowGetSize(ctx)
	_, _ = a.settings.Update(func(s *settings.Settings) {
		if maximised && s.Window != nil {
			s.Window.Maximised = true // keep the restored size for next time
			return
		}
		s.Window = &settings.WindowBounds{Width: w, Height: h, Maximised: maximised}
	})
	return false
}

// --- launch & app info ---------------------------------------------------------

// GetLaunchIntent says what the window was opened for.
func (a *App) GetLaunchIntent() launch.Intent {
	in := a.intent
	if in.Extra == nil {
		in.Extra = []string{}
	}
	if in.Source != "" {
		if abs, err := filepath.Abs(in.Source); err == nil {
			in.Source = abs
		}
	}
	if in.Target != "" {
		if abs, err := filepath.Abs(in.Target); err == nil {
			in.Target = abs
		}
	}
	return in
}

// Version is the app version (set at build time).
func (a *App) Version() string { return Version }

// Quit closes the app.
func (a *App) Quit() {
	if !a.headless {
		runtime.Quit(a.ctx)
	}
}

// --- templates -----------------------------------------------------------------

// Inspect describes a folder: template or plain, size, validation issues.
func (a *App) Inspect(dir string) (*ft.Inspection, error) {
	return ft.Inspect(dir)
}

// Issue is one validation problem; Param is "" when not tied to a parameter.
type Issue struct {
	Param   string `json:"param"`
	Message string `json:"message"`
}

// Validate runs the engine's descriptor checks on an unsaved descriptor.
func (a *App) Validate(desc ft.Template) []Issue {
	out := []Issue{}
	for _, err := range desc.Validate() {
		var verr *ft.ValidationError
		if errors.As(err, &verr) {
			out = append(out, Issue{Param: verr.Param, Message: verr.Err.Error()})
		} else {
			out = append(out, Issue{Message: err.Error()})
		}
	}
	return out
}

// SaveTemplate writes desc to dir/.ft/template.json.
func (a *App) SaveTemplate(dir string, desc ft.Template) error {
	if err := ft.Save(&desc, dir); err != nil {
		return err
	}
	a.RecordRecent(dir)
	return nil
}

// bound returns the template at dir, or desc bound to dir when the editor
// passes unsaved changes.
func bound(dir string, desc *ft.Template) (*ft.Template, error) {
	if desc != nil {
		return ft.Bind(*desc, dir)
	}
	return ft.Load(dir)
}

// Preview dry-runs generation into target.
func (a *App) Preview(dir string, desc *ft.Template, values map[string]string, target string) (*ft.PreviewResult, error) {
	tpl, err := bound(dir, desc)
	if err != nil {
		return nil, err
	}
	if target == "" {
		target = ft.ResolveDefaultTarget(tpl)
	}
	return ft.PreviewAt(tpl, target, values, nil, nil)
}

// Rendered is one .ft$ file before and after token replacement.
type Rendered struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// RenderFile fills one .ft$ file's tokens without writing anything.
func (a *App) RenderFile(dir string, desc *ft.Template, sourceRel string, values map[string]string) (*Rendered, error) {
	tpl, err := bound(dir, desc)
	if err != nil {
		return nil, err
	}
	before, after, err := ft.RenderFile(tpl, sourceRel, values, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Rendered{Before: before, After: after}, nil
}

// Tree lists the template folder's source tree.
func (a *App) Tree(dir string) ([]ft.TreeEntry, error) { return ft.Tree(dir) }

// Scan finds tokens and authoring issues, against unsaved edits if given.
func (a *App) Scan(dir string, desc *ft.Template) (*ft.ScanResult, error) {
	return ft.Scan(dir, desc, nil)
}

// TestMatch lists the names a pattern would rename.
func (a *App) TestMatch(dir, pattern, replacement string) ([]ft.NameHit, error) {
	return ft.TestMatch(dir, pattern, replacement, nil)
}

// SetContentProcessing adds or strips a file's .ft$ suffix.
func (a *App) SetContentProcessing(dir, rel string, on bool) (string, error) {
	return ft.SetContentProcessing(dir, rel, on)
}

// --- generating ----------------------------------------------------------------

// GenerateRequest is what the generate view submits.
type GenerateRequest struct {
	Dir      string            `json:"dir"`
	Target   string            `json:"target"`
	Values   map[string]string `json:"values"`
	Conflict string            `json:"conflict"`
}

// Progress is emitted as "generate:progress" while generating.
type Progress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Path  string `json:"path"`
}

// Generate creates the output folder, emitting progress events. Only one
// generation runs at a time; CancelGenerate stops it.
func (a *App) Generate(req GenerateRequest) (*ft.Result, error) {
	policy, err := ft.ParseConflictPolicy(req.Conflict)
	if err != nil {
		return nil, err
	}
	tpl, err := ft.Load(req.Dir)
	if err != nil {
		return nil, err
	}
	target := req.Target
	if target == "" {
		target = ft.ResolveDefaultTarget(tpl)
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	if a.genCancel != nil {
		a.mu.Unlock()
		cancel()
		return nil, errors.New("a generation is already running")
	}
	a.genCancel = cancel
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		a.genCancel = nil
		a.mu.Unlock()
	}()

	var last time.Time
	progress := func(done, total int, rel string) {
		// Throttle: the UI needs a smooth bar, not an event per file.
		if done < total && time.Since(last) < 40*time.Millisecond {
			return
		}
		last = time.Now()
		if !a.headless {
			runtime.EventsEmit(a.ctx, "generate:progress", Progress{Done: done, Total: total, Path: rel})
		}
	}

	res, err := ft.GenerateContext(ctx, tpl, target, req.Values, nil, &ft.Options{Conflict: policy, Progress: progress})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return res, errors.New("cancelled — files created so far were left in place")
		}
		return res, err
	}
	now := time.Now()
	_, _ = a.settings.Update(func(s *settings.Settings) {
		s.TouchRecent(tpl.Dir(), now)
		s.TouchTarget(target)
	})
	return res, nil
}

// CancelGenerate stops a running generation.
func (a *App) CancelGenerate() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.genCancel != nil {
		a.genCancel()
	}
}

// --- files & folders -------------------------------------------------------------

// PickFolder shows the native folder picker; "" when cancelled.
func (a *App) PickFolder(title, initial string) (string, error) {
	if a.headless {
		return "", errors.New("the folder picker needs the desktop app")
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title, DefaultDirectory: existingDir(initial)})
}

// existingDir walks up to the nearest folder that exists, so the picker can
// start near a destination that hasn't been created yet.
func existingDir(p string) string {
	for p != "" {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return ""
}

// OpenFolder opens a folder in the file manager.
func (a *App) OpenFolder(path string) error { return shell.OpenFolder(path) }

// RevealPath shows a file or folder selected in its parent folder.
func (a *App) RevealPath(path string) error { return shell.Reveal(path) }

// --- library & settings ----------------------------------------------------------

// ListLibrary returns the templates for the home screen.
func (a *App) ListLibrary() []library.Entry {
	s := a.settings.Get()
	in := library.Input{LibraryFolders: s.LibraryFolders, Pinned: s.Pinned, Recents: s.Recents}
	if s.ShowSamples {
		in.SamplesDir = a.samples
	}
	return library.List(in)
}

// RecordRecent notes that the template at dir was opened.
func (a *App) RecordRecent(dir string) {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	now := time.Now()
	_, _ = a.settings.Update(func(s *settings.Settings) { s.TouchRecent(dir, now) })
}

// RemoveRecent forgets a recent template.
func (a *App) RemoveRecent(path string) error {
	_, err := a.settings.Update(func(s *settings.Settings) { s.RemoveRecent(path) })
	return err
}

// SetPinned pins or unpins a template on the home screen.
func (a *App) SetPinned(path string, pinned bool) error {
	_, err := a.settings.Update(func(s *settings.Settings) { s.SetPinned(path, pinned) })
	return err
}

// GetSettings returns the current settings.
func (a *App) GetSettings() settings.Settings { return a.settings.Get() }

// SaveSettings stores the settings page's values.
func (a *App) SaveSettings(s settings.Settings) (settings.Settings, error) {
	return a.settings.Replace(s)
}

// SamplesFolder is where the bundled samples live ("" if not installed).
func (a *App) SamplesFolder() string { return a.samples }

// --- shell integration ------------------------------------------------------------

// ShellStatus reports which Explorer integrations point at this app. When
// there is no app Explorer could launch (a debug build with no release build
// beside it), everything reads as off.
func (a *App) ShellStatus() (map[shell.Feature]bool, error) {
	in, err := integration()
	if err != nil {
		off := map[shell.Feature]bool{}
		for _, f := range shell.Features {
			off[f] = false
		}
		return off, nil
	}
	return in.Status(), nil
}

// SetShellFeature turns one Explorer integration on or off.
func (a *App) SetShellFeature(feature shell.Feature, on bool) (map[shell.Feature]bool, error) {
	in, err := integration()
	if err != nil {
		return nil, err
	}
	if err := in.Set(feature, on); err != nil {
		return nil, err
	}
	return in.Status(), nil
}

// integration targets the exe Explorer should launch. That is normally the
// running app, but never a debug build: it is a console program that only
// starts under the debugger (the Wails dev tag looks for the frontend relative
// to the working directory), so Explorer would flash a terminal and nothing
// more. A debug build registers the release FolderTemplates.exe beside it.
func integration() (*shell.Integration, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if devBuild {
		release := filepath.Join(filepath.Dir(exe), "FolderTemplates.exe")
		if info, err := os.Stat(release); err != nil || info.IsDir() {
			return nil, errors.New("this is a debug build, which Explorer can't start; build the app with `wails build` (or scripts/build.ps1) and turn this on again")
		}
		exe = release
	}
	return shell.For(exe)
}
