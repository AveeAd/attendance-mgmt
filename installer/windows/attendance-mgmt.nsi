; NSIS installer for attendance-mgmt (Windows).
;
; Unsigned, per-user install — no admin/UAC prompt, true one-click.
; Auto-starts via a shortcut in the current user's Startup folder — this
; is the standard "run at login" mechanism for a per-user app and needs
; no schtasks.exe command-line quoting (which is genuinely hard to get
; right and hard to verify without a real Windows machine to test on).
;
; Expects the raw binary at "attendance-mgmt.exe" next to this script
; (CI copies/renames the windows/amd64 build there before running
; makensis). Build with:
;   makensis installer\windows\attendance-mgmt.nsi

!define APP_NAME "attendance-mgmt"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

; Version can be overridden at build time: makensis -DVERSION=v1.2.3 ...
!ifndef VERSION
  !define VERSION "dev"
!endif

Name "Attendance Management System"
OutFile "attendance-mgmt-windows-amd64-setup.exe"
InstallDir "$LOCALAPPDATA\${APP_NAME}"
RequestExecutionLevel user
SetCompressor /SOLID lzma

Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

Function .onInit
  ; If already installed, stop the running app before overwriting it —
  ; this makes re-running the installer a valid (if secondary) update path.
  nsExec::ExecToLog 'taskkill /IM attendance-mgmt.exe /F'
  Pop $0
FunctionEnd

Section "Install"
  SetOutPath "$INSTDIR"
  File "attendance-mgmt.exe"
  CreateDirectory "$INSTDIR\data"

  ; Small launcher so we can set ATTENDANCE_DB_PATH and start the server
  ; minimized (no console window in your face), and to keep data\ out of
  ; the install root so an exe-only update never touches it.
  FileOpen $0 "$INSTDIR\run.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 'set ATTENDANCE_DB_PATH=%~dp0data\attendance.db$\r$\n'
  FileWrite $0 'start /min "" "%~dp0attendance-mgmt.exe"$\r$\n'
  FileClose $0

  WriteRegStr HKCU "Software\${APP_NAME}" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "Software\${APP_NAME}" "Version" "${VERSION}"

  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "Attendance Management System"
  WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1

  ; Auto-start on every login for the current user. $SMSTARTUP is NSIS's
  ; built-in path for the current user's Startup folder.
  CreateShortcut "$SMSTARTUP\Attendance Management System.lnk" "$INSTDIR\run.bat" "" "" 0 SW_SHOWMINIMIZED

  ; Start it now too, so you don't have to log off/on to use it.
  Exec '"$INSTDIR\run.bat"'

  MessageBox MB_OK "Attendance Management System installed and started.$\r$\n$\r$\nOpen http://localhost:8080 in your browser.$\r$\n$\r$\nIt will now start automatically every time you log in."
SectionEnd

Section "Uninstall"
  nsExec::ExecToLog 'taskkill /IM attendance-mgmt.exe /F'
  Pop $0

  Delete "$SMSTARTUP\Attendance Management System.lnk"
  Delete "$INSTDIR\attendance-mgmt.exe"
  Delete "$INSTDIR\run.bat"
  Delete "$INSTDIR\uninstall.exe"
  ; Deliberately NOT deleting $INSTDIR\data — leave attendance/payroll data
  ; in place unless the user removes it manually.

  DeleteRegKey HKCU "${UNINST_KEY}"
  DeleteRegKey HKCU "Software\${APP_NAME}"
SectionEnd
