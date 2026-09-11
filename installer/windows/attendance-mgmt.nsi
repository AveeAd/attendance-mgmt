; NSIS installer for attendance-mgmt (Windows).
;
; Unsigned, per-user install — no admin/UAC prompt, true one-click.
; Registers a Task Scheduler entry (ONLOGON trigger) so the app starts
; automatically every time the user logs in, matching the product spec's
; decided auto-start mechanism for Windows.
;
; Expects the raw binary at "attendance-mgmt.exe" next to this script
; (CI copies/renames the windows/amd64 build there before running
; makensis). Build with:
;   makensis installer\windows\attendance-mgmt.nsi

!define APP_NAME "attendance-mgmt"
!define TASK_NAME "AttendanceMgmt"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

; Version can be overridden at build time: makensis /DVERSION=v1.2.3 ...
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
  nsExec::ExecToLog 'schtasks /End /TN "${TASK_NAME}"'
  Pop $0
  nsExec::ExecToLog 'taskkill /IM attendance-mgmt.exe /F'
  Pop $0
FunctionEnd

Section "Install"
  SetOutPath "$INSTDIR"
  File "attendance-mgmt.exe"
  CreateDirectory "$INSTDIR\data"

  ; Small launcher so the scheduled task can set ATTENDANCE_DB_PATH without
  ; fighting schtasks' command-line quoting, and keeps data\ out of the
  ; install root so an exe-only update never touches it.
  FileOpen $0 "$INSTDIR\run.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 'set ATTENDANCE_DB_PATH=%~dp0data\attendance.db$\r$\n'
  FileWrite $0 'start "" "%~dp0attendance-mgmt.exe"$\r$\n'
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

  ; Auto-start on every login for the current user, no elevation needed.
  nsExec::ExecToLog 'schtasks /Create /TN "${TASK_NAME}" /TR "\"$INSTDIR\run.bat\"" /SC ONLOGON /RL LIMITED /F'
  Pop $0

  ; schtasks /Run triggers the task via the Task Scheduler service and
  ; returns immediately — it does not wait for the (long-running) app to
  ; exit, unlike nsExec on the batch file directly.
  nsExec::ExecToLog 'schtasks /Run /TN "${TASK_NAME}"'
  Pop $0

  MessageBox MB_OK "Attendance Management System installed and started.$\r$\n$\r$\nIt will now start automatically every time you log in."
SectionEnd

Section "Uninstall"
  nsExec::ExecToLog 'schtasks /Delete /TN "${TASK_NAME}" /F'
  Pop $0
  nsExec::ExecToLog 'taskkill /IM attendance-mgmt.exe /F'
  Pop $0

  Delete "$INSTDIR\attendance-mgmt.exe"
  Delete "$INSTDIR\run.bat"
  Delete "$INSTDIR\uninstall.exe"
  ; Deliberately NOT deleting $INSTDIR\data — leave attendance/payroll data
  ; in place unless the user removes it manually.

  DeleteRegKey HKCU "${UNINST_KEY}"
  DeleteRegKey HKCU "Software\${APP_NAME}"
SectionEnd
