package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/harvey-withington/folder-templates-2.0/app/internal/launch"
	"github.com/harvey-withington/folder-templates-2.0/app/internal/settings"
)

// serveUI runs the app's UI in a browser instead of a window: the embedded
// frontend is served over HTTP on addr (loopback only), and a small injected
// script turns its window.go.main.App.* calls into requests that invoke the
// same Go methods. Nothing is shown on the desktop — scripts/screenshots.mjs
// points headless Edge at it. args are the usual launch arguments.
//
//	FolderTemplates.exe --serve-ui 127.0.0.1:7931 [-edit -sourceFolder <dir>]
func serveUI(addr string, args []string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("--serve-ui only binds to loopback addresses, not %q", host)
	}

	path, err := settings.DefaultPath()
	if err != nil {
		return err
	}
	store, err := settings.Open(path)
	if err != nil {
		return err
	}
	app := NewApp(launch.Parse(args), store, findSamples(), 0)
	app.ctx = context.Background()
	app.headless = true

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return err
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return err
	}
	index = bytes.Replace(index, []byte("<head>"), []byte("<head><script src=\"/__bridge.js\"></script>"), 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/__bridge.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		fmt.Fprint(w, strings.ReplaceAll(bridgeJS, "__TOKEN__", token))
	})
	mux.HandleFunc("/__call/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("X-FT-Token") != token {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		callMethod(w, r, app, strings.TrimPrefix(r.URL.Path, "/__call/"))
	})
	files := http.FileServer(http.FS(dist))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(index)
			return
		}
		files.ServeHTTP(w, r)
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("serving the UI on http://%s\n", ln.Addr())
	return http.Serve(ln, mux)
}

// callMethod invokes app.<name>(args...) with the JSON array body as the
// arguments and writes the first result as JSON, or the error as a 500.
func callMethod(w http.ResponseWriter, r *http.Request, app *App, name string) {
	m := reflect.ValueOf(app).MethodByName(name)
	if !m.IsValid() {
		http.Error(w, "no method "+name, http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var raw []json.RawMessage
	if len(body) > 0 {
		if err := json.Unmarshal(body, &raw); err != nil {
			http.Error(w, "arguments must be a JSON array", http.StatusBadRequest)
			return
		}
	}
	t := m.Type()
	if len(raw) != t.NumIn() {
		http.Error(w, fmt.Sprintf("%s takes %d arguments, got %d", name, t.NumIn(), len(raw)), http.StatusBadRequest)
		return
	}
	in := make([]reflect.Value, len(raw))
	for i := range raw {
		p := reflect.New(t.In(i))
		if err := json.Unmarshal(raw[i], p.Interface()); err != nil {
			http.Error(w, fmt.Sprintf("argument %d: %v", i, err), http.StatusBadRequest)
			return
		}
		in[i] = p.Elem()
	}
	out := m.Call(in)
	if n := len(out); n > 0 && t.Out(n-1) == reflect.TypeFor[error]() {
		if e := out[n-1]; !e.IsNil() {
			http.Error(w, e.Interface().(error).Error(), http.StatusInternalServerError)
			return
		}
		out = out[:n-1]
	}
	w.Header().Set("Content-Type", "application/json")
	if len(out) == 0 {
		w.Write([]byte("null"))
		return
	}
	_ = json.NewEncoder(w).Encode(out[0].Interface())
}

// bridgeJS stands in for the Wails runtime in the browser: bound methods go
// over HTTP (rejecting with the Go error text, as Wails does), and the few
// runtime calls the UI makes become harmless browser equivalents.
const bridgeJS = `(() => {
  const token = '__TOKEN__'
  const call = (name) => async (...args) => {
    const res = await fetch('/__call/' + name, {
      method: 'POST',
      headers: { 'X-FT-Token': token, 'Content-Type': 'application/json' },
      body: JSON.stringify(args),
    })
    const text = await res.text()
    if (!res.ok) throw text
    return text ? JSON.parse(text) : null
  }
  window.go = { main: { App: new Proxy({}, { get: (_, name) => call(String(name)) }) } }
  window.runtime = {
    EventsOn: () => () => {},
    OnFileDrop: () => {},
    OnFileDropOff: () => {},
    WindowSetTitle: (title) => { document.title = title },
  }
})()
`
