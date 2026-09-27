// Package cli implements the ft command: modern subcommands plus a
// compatibility mode that accepts the C# FolderTemplates.Console flags.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	ft "github.com/harvey-withington/foldertemplate"
)

// Exit codes.
const (
	ExitOK        = 0
	ExitUsage     = 1 // bad arguments, missing paths
	ExitTemplate  = 2 // template missing, unparsable or invalid
	ExitConflict  = 3 // output already exists under the refuse policy
	ExitIO        = 4 // anything else going wrong while generating
	ExitCancelled = 130
)

// Env is everything Run touches outside its arguments, so tests can drive it.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Interactive reports whether Stdin is a terminal a person can answer.
	Interactive bool
	// WaitKey blocks until a key is pressed (v1 mode without -nowait).
	WaitKey func()
	// Ctx is cancelled on Ctrl+C.
	Ctx context.Context
	// Newline is written between lines of v1 JSON output ("\r\n" on
	// Windows, like the C# app).
	Newline string
	// Version is printed by `ft version`.
	Version string

	in *bufio.Reader
}

func (e *Env) reader() *bufio.Reader {
	if e.in == nil {
		e.in = bufio.NewReader(e.Stdin)
	}
	return e.in
}

// readLine reads one line; ok is false at end of input.
func (e *Env) readLine() (string, bool) {
	line, err := e.reader().ReadString('\n')
	if err != nil && line == "" {
		return "", false
	}
	return strings.TrimRight(line, "\r\n"), true
}

func (e *Env) printf(format string, a ...any) { fmt.Fprintf(e.Stdout, format, a...) }
func (e *Env) errorf(format string, a ...any) { fmt.Fprintf(e.Stderr, format, a...) }

var modernCommands = map[string]func(*Env, []string) int{
	"generate": cmdGenerate,
	"params":   cmdParams,
	"info":     cmdInfo,
	"validate": cmdValidate,
	"scan":     cmdScan,
}

// Run executes ft with args (excluding the program name) and returns the
// process exit code.
func Run(args []string, env *Env) int {
	if env.Ctx == nil {
		env.Ctx = context.Background()
	}
	if env.Newline == "" {
		env.Newline = "\n"
	}
	if len(args) == 0 {
		usage(env.Stdout)
		return ExitUsage
	}
	switch args[0] {
	case "help", "--help", "-h", "/?":
		usage(env.Stdout)
		return ExitOK
	case "version", "--version":
		env.printf("ft %s\n", env.Version)
		return ExitOK
	}
	if cmd, ok := modernCommands[args[0]]; ok {
		return cmd(env, args[1:])
	}
	// Anything else is the C# console's syntax: single-dash flags, or a bare
	// template path (which that app took as -sourceFolder).
	if strings.HasPrefix(args[0], "--") {
		env.errorf("ft: unknown option %s\n\n", args[0])
		usage(env.Stderr)
		return ExitUsage
	}
	return runV1(env, args)
}

func usage(w io.Writer) {
	fmt.Fprint(w, `ft — generate folders from Folder Templates

Usage:
  ft generate <template> [--target DIR] [--set name=value]... [--on-conflict refuse|merge|overwrite]
                         [--template-file FILE] [--dry-run] [--no-prompt] [--json]
  ft params   <template> [--all] [--json]     list the parameters a template asks for
  ft info     <template> [--json]             name, description, default target, size
  ft validate <template> [--json]             check the template descriptor
  ft scan     <template> [--json]             find tokens and authoring mistakes
  ft version

<template> is a folder containing .ft/template.json.

Folder Templates 1.0 console syntax is also accepted, e.g.
  ft -sourceFolder <template> -targetFolder <dir> -name value -nowait -noprompt
  ft -sourceFolder <template> -listParams json

Exit codes: 0 ok, 1 usage, 2 template invalid, 3 target exists, 4 I/O error.
`)
}

// exitFor maps an engine error to an exit code.
func exitFor(err error) int {
	var verr *ft.ValidationError
	switch {
	case err == nil:
		return ExitOK
	case errors.Is(err, context.Canceled):
		return ExitCancelled
	case errors.Is(err, ft.ErrTargetExists):
		return ExitConflict
	case errors.As(err, &verr), errors.Is(err, ft.ErrNotATemplate),
		errors.Is(err, ft.ErrBinaryContent), errors.Is(err, ft.ErrContentTooLarge):
		return ExitTemplate
	}
	return ExitIO
}
