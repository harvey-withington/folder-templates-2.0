//go:build windows

package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Integration installs and removes the shell entries for one executable.
// Everything lives under the current user (the SendTo folder and
// HKCU\Software\Classes), so no admin rights are needed.
type Integration struct {
	// Exe is the program the entries launch.
	Exe string
	// SendToDir is the user's SendTo folder.
	SendToDir string
	// ClassesKey is the HKCU subkey holding shell verbs.
	ClassesKey string
}

// Default targets the running executable and the real user locations.
func Default() (*Integration, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	sendTo, err := windows.KnownFolderPath(windows.FOLDERID_SendTo, 0)
	if err != nil {
		return nil, err
	}
	return &Integration{Exe: exe, SendToDir: sendTo, ClassesKey: `Software\Classes`}, nil
}

// Status reports which features currently point at this executable. A 1.0
// shortcut with the same name but another target counts as off.
func (in *Integration) Status() map[Feature]bool {
	return map[Feature]bool{
		SendToProcess:  in.shortcutIsOurs(sendToProcessName),
		SendToEdit:     in.shortcutIsOurs(sendToEditName),
		FolderMenu:     in.verbIsOurs(`Directory\shell\`+verbGenerate) && in.verbIsOurs(`Directory\shell\`+verbEdit),
		BackgroundMenu: in.verbIsOurs(`Directory\Background\shell\` + verbNewHere),
	}
}

// Set turns one feature on or off.
func (in *Integration) Set(f Feature, on bool) error {
	switch f {
	case SendToProcess:
		return in.setShortcut(sendToProcessName, "", on)
	case SendToEdit:
		return in.setShortcut(sendToEditName, "-edit -sourceFolder", on)
	case FolderMenu:
		if err := in.setVerb(`Directory\shell\`+verbGenerate, labelGenerate, `-sourceFolder "%1"`, on); err != nil {
			return err
		}
		return in.setVerb(`Directory\shell\`+verbEdit, labelEdit, `-edit -sourceFolder "%1"`, on)
	case BackgroundMenu:
		return in.setVerb(`Directory\Background\shell\`+verbNewHere, labelNewHere, `--pick-template --target "%V"`, on)
	}
	return fmt.Errorf("unknown shell feature %q", f)
}

// RemoveAll turns every feature off (the uninstaller's cleanup).
func (in *Integration) RemoveAll() error {
	var errs []error
	for _, f := range Features {
		if err := in.Set(f, false); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// --- registry verbs --------------------------------------------------------------

func (in *Integration) command(args string) string {
	return `"` + in.Exe + `" ` + args
}

func (in *Integration) setVerb(rel, label, args string, on bool) error {
	path := in.ClassesKey + `\` + rel
	if !on {
		return deleteTree(registry.CURRENT_USER, path)
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetStringValue("", label); err != nil {
		return err
	}
	if err := k.SetStringValue("Icon", `"`+in.Exe+`",0`); err != nil {
		return err
	}
	cmd, _, err := registry.CreateKey(registry.CURRENT_USER, path+`\command`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer cmd.Close()
	return cmd.SetStringValue("", in.command(args))
}

func (in *Integration) verbIsOurs(rel string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, in.ClassesKey+`\`+rel+`\command`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("")
	return err == nil && strings.HasPrefix(strings.ToLower(v), strings.ToLower(`"`+in.Exe+`"`))
}

// deleteTree removes a key and its subkeys; a missing key is not an error.
func deleteTree(root registry.Key, path string) error {
	k, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	subs, err := k.ReadSubKeyNames(-1)
	k.Close()
	if err != nil {
		return err
	}
	for _, s := range subs {
		if err := deleteTree(root, path+`\`+s); err != nil {
			return err
		}
	}
	err = registry.DeleteKey(root, path)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}

// --- SendTo shortcuts ----------------------------------------------------------------

func (in *Integration) linkPath(name string) string {
	return filepath.Join(in.SendToDir, name+".lnk")
}

func (in *Integration) setShortcut(name, args string, on bool) error {
	link := in.linkPath(name)
	if !on {
		if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return withShortcut(link, func(sc *ole.IDispatch) error {
		for prop, val := range map[string]string{
			"TargetPath":       in.Exe,
			"Arguments":        args,
			"WorkingDirectory": filepath.Dir(in.Exe),
			"IconLocation":     in.Exe + ",0",
			"Description":      "Folder Templates",
		} {
			if _, err := oleutil.PutProperty(sc, prop, val); err != nil {
				return fmt.Errorf("set %s: %w", prop, err)
			}
		}
		_, err := oleutil.CallMethod(sc, "Save")
		return err
	})
}

func (in *Integration) shortcutIsOurs(name string) bool {
	link := in.linkPath(name)
	if _, err := os.Stat(link); err != nil {
		return false
	}
	ours := false
	_ = withShortcut(link, func(sc *ole.IDispatch) error {
		v, err := oleutil.GetProperty(sc, "TargetPath")
		if err != nil {
			return err
		}
		ours = strings.EqualFold(filepath.Clean(v.ToString()), filepath.Clean(in.Exe))
		return nil
	})
	return ours
}

// withShortcut opens (or starts) a .lnk through WScript.Shell, the same COM
// object the C# app used, on a locked OS thread as COM requires.
func withShortcut(link string, fn func(*ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		var oleErr *ole.OleError
		// S_FALSE: already initialized on this thread — fine.
		if !errors.As(err, &oleErr) || oleErr.Code() != 1 {
			return err
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unknown.Release()
	wsh, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wsh.Release()
	res, err := oleutil.CallMethod(wsh, "CreateShortcut", link)
	if err != nil {
		return err
	}
	sc := res.ToIDispatch()
	defer sc.Release()
	return fn(sc)
}
