!include "MUI2.nsh"
!include "FileFunc.nsh"

Name "3D Job Desk Printer Bridge"
OutFile "${OUTFILE}"
Unicode True
InstallDir "$LOCALAPPDATA\Programs\3D Job Desk Bridge"
InstallDirRegKey HKCU "Software\Coex\3D Job Desk Bridge" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma
BrandingText "Coex · 3D Job Desk"

!define MUI_ICON "${ICON}"
!define MUI_UNICON "${ICON}"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "3D Job Desk Printer Bridge"
!define MUI_WELCOMEPAGE_TEXT "This installs a small program on this computer so 3D Job Desk can read printer status on your shop LAN.$\r$\n$\r$\nIt only makes outbound HTTPS calls to app.3djobdesk.com. It does not open a port or send print jobs.$\r$\n$\r$\nClick Next to continue."
!define MUI_FINISHPAGE_RUN "$INSTDIR\3d-job-desk-bridge.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Open Printer Bridge now"
!define MUI_FINISHPAGE_NOAUTOCLOSE

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Printer Bridge (required)" SecApp
  SectionIn RO
  SetOutPath "$INSTDIR"
  File /oname=3d-job-desk-bridge.exe "${BRIDGE_EXE}"
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "Software\Coex\3D Job Desk Bridge" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "DisplayName" "3D Job Desk Printer Bridge"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "DisplayIcon" "$INSTDIR\3d-job-desk-bridge.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "Publisher" "Coex"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "DisplayVersion" "1.2.0"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "HelpLink" "https://app.3djobdesk.com"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "URLInfoAbout" "https://3djobdesk.com"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "NoRepair" 1
  CreateDirectory "$SMPROGRAMS\3D Job Desk"
  CreateShortCut "$SMPROGRAMS\3D Job Desk\Printer Bridge.lnk" "$INSTDIR\3d-job-desk-bridge.exe" "" "$INSTDIR\3d-job-desk-bridge.exe" 0
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge" "EstimatedSize" "$0"
SectionEnd

Section "Start with Windows" SecAutoStart
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "3DJobDeskBridge" '"$INSTDIR\3d-job-desk-bridge.exe" --tray'
SectionEnd

Section /o "Desktop shortcut" SecDesktop
  CreateShortCut "$DESKTOP\3D Job Desk Bridge.lnk" "$INSTDIR\3d-job-desk-bridge.exe" "" "$INSTDIR\3d-job-desk-bridge.exe" 0
SectionEnd

LangString DESC_SecApp ${LANG_ENGLISH} "The Printer Bridge program, Start Menu shortcut, and uninstaller."
LangString DESC_SecAutoStart ${LANG_ENGLISH} "Start Printer Bridge when you sign in to Windows. You can also change this later in the app."
LangString DESC_SecDesktop ${LANG_ENGLISH} "Put a shortcut on the desktop."

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecApp} $(DESC_SecApp)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAutoStart} $(DESC_SecAutoStart)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} $(DESC_SecDesktop)
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "Uninstall"
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "3DJobDeskBridge"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\3DJobDeskBridge"
  DeleteRegKey HKCU "Software\Coex\3D Job Desk Bridge"
  Delete "$SMPROGRAMS\3D Job Desk\Printer Bridge.lnk"
  RMDir "$SMPROGRAMS\3D Job Desk"
  Delete "$DESKTOP\3D Job Desk Bridge.lnk"
  Delete "$INSTDIR\3d-job-desk-bridge.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
