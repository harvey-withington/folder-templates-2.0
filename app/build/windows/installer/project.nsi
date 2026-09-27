Unicode true

####
## Folder Templates installer — per-user, no administrator rights.
##
## Built by `wails build -nsis` (see scripts/build.ps1), which regenerates
## wails_tools.nsh and passes ARG_WAILS_AMD64_BINARY. ft.exe is built by the
## script into build\ft\ before Wails runs; samples come from the repo.
####

!define REQUEST_EXECUTION_LEVEL "user"
!define UNINST_KEY_NAME "FolderTemplates"

!include "wails_tools.nsh"
!include "MUI2.nsh"
!include "Sections.nsh"
!include "WordFunc.nsh"
!include "WinMessages.nsh"

!define USER_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}"
!define FT_EXE "..\..\ft\ft.exe"
!define SAMPLES_DIR "..\..\..\..\samples"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
InstallDirRegKey HKCU "${USER_UNINST_KEY}" "InstallLocation"
ShowInstDetails show

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING
!define MUI_COMPONENTSPAGE_SMALLDESC
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Open Folder Templates"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Function .onInit
    !insertmacro wails.checkArchitecture
FunctionEnd

Section "Folder Templates" SecCore
    SectionIn RO
    SetShellVarContext current
    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    !insertmacro wails.files
    File "${FT_EXE}"
    File "..\..\..\..\LICENSE"

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    WriteUninstaller "$INSTDIR\uninstall.exe"
    WriteRegStr HKCU "${USER_UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKCU "${USER_UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKCU "${USER_UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKCU "${USER_UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKCU "${USER_UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKCU "${USER_UNINST_KEY}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "${USER_UNINST_KEY}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"
    WriteRegDWORD HKCU "${USER_UNINST_KEY}" "NoModify" 1
    WriteRegDWORD HKCU "${USER_UNINST_KEY}" "NoRepair" 1
SectionEnd

Section "Sample templates" SecSamples
    SetOutPath "$INSTDIR\samples"
    File /r "${SAMPLES_DIR}\*.*"
    SetOutPath $INSTDIR
SectionEnd

Section "Send to shortcuts" SecSendTo
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" --shell-install sendToProcess,sendToEdit'
SectionEnd

Section "Explorer right-click menus" SecMenus
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" --shell-install folderMenu,backgroundMenu'
SectionEnd

Section "Add ft to PATH" SecPath
    ReadRegStr $0 HKCU "Environment" "Path"
    ${WordFind} "$0" "$INSTDIR" "E+1{" $1
    IfErrors 0 path_done
        StrCmp $0 "" 0 +3
            WriteRegExpandStr HKCU "Environment" "Path" "$INSTDIR"
            Goto path_notify
        WriteRegExpandStr HKCU "Environment" "Path" "$0;$INSTDIR"
    path_notify:
        SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000
    path_done:
SectionEnd

Section /o "Desktop shortcut" SecDesktop
    SetShellVarContext current
    CreateShortcut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "The app and the ft command-line tool."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecSamples} "Example templates, including one that shows every feature."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecSendTo} "Folder Template - Process and Folder Template - Edit in Explorer's Send to menu. Replaces Folder Templates 1.0's shortcuts."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecMenus} "Right-click a folder to generate from it or edit it; right-click inside a folder for New from template here."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecPath} "Lets you run ft from any terminal."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "A shortcut on your desktop."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "uninstall"
    SetShellVarContext current
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" --shell-uninstall'

    ReadRegStr $0 HKCU "Environment" "Path"
    ${un.WordReplace} "$0" ";$INSTDIR" "" "+" $1
    ${un.WordReplace} "$1" "$INSTDIR;" "" "+" $1
    ${un.WordReplace} "$1" "$INSTDIR" "" "+" $1
    StrCmp $0 $1 +3
        WriteRegExpandStr HKCU "Environment" "Path" "$1"
        SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # WebView2 data
    RMDir /r $INSTDIR
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    DeleteRegKey HKCU "${USER_UNINST_KEY}"
SectionEnd
