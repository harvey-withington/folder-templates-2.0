package shell

// Feature is one piece of Windows shell integration the user can toggle.
type Feature string

const (
	// SendToProcess is the "Folder Template - Process" SendTo shortcut:
	// send a template folder to it to generate from it.
	SendToProcess Feature = "sendToProcess"
	// SendToEdit is the "Folder Template - Edit" SendTo shortcut.
	SendToEdit Feature = "sendToEdit"
	// FolderMenu adds "Generate from this template" and "Edit folder
	// template" to a folder's right-click menu.
	FolderMenu Feature = "folderMenu"
	// BackgroundMenu adds "New from template here…" to the right-click menu
	// of a folder's empty space.
	BackgroundMenu Feature = "backgroundMenu"
)

// Features lists every toggle, in display order.
var Features = []Feature{SendToProcess, SendToEdit, FolderMenu, BackgroundMenu}

// SendTo shortcut names — identical to Folder Templates 1.0's, so turning
// them on over a 1.0 install replaces its shortcuts instead of adding more.
const (
	sendToProcessName = "Folder Template - Process"
	sendToEditName    = "Folder Template - Edit"
)

// Registry verb keys under <classes>\Directory\shell and
// <classes>\Directory\Background\shell.
const (
	verbGenerate = "FolderTemplates.Generate"
	verbEdit     = "FolderTemplates.Edit"
	verbNewHere  = "FolderTemplates.NewHere"
)

// Labels shown in Explorer. Explorer menus aren't localized through the app's
// catalogue, so they live here.
const (
	labelGenerate = "Generate from this template"
	labelEdit     = "Edit folder template"
	labelNewHere  = "New from template here…"
)
