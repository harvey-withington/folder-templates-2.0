// Command ft generates folders from Folder Templates. It also accepts the
// Folder Templates 1.0 console syntax (FolderTemplates.Console.exe).
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"

	"golang.org/x/term"

	"github.com/harvey-withington/folder-templates/internal/cli"
)

// Version is set at build time with -ldflags "-X main.Version=…".
var Version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	newline := "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}
	stdinFd := int(os.Stdin.Fd())
	env := &cli.Env{
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Interactive: term.IsTerminal(stdinFd),
		WaitKey:     func() { waitKey(stdinFd) },
		Ctx:         ctx,
		Newline:     newline,
		Version:     Version,
	}
	os.Exit(cli.Run(os.Args[1:], env))
}

// waitKey returns after one key press, like .NET's Console.ReadKey. Without a
// terminal there is no one to press a key, so it returns at once.
func waitKey(fd int) {
	if !term.IsTerminal(fd) {
		return
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, state)
	var b [1]byte
	_, _ = os.Stdin.Read(b[:])
}
