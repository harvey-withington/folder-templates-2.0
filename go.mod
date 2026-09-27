module github.com/harvey-withington/folder-templates

go 1.26.0

require (
	github.com/harvey-withington/foldertemplate v0.0.0
	golang.org/x/term v0.46.0
)

require (
	github.com/dlclark/regexp2 v1.11.5 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

replace github.com/harvey-withington/foldertemplate => ./engine
