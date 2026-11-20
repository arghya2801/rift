Unicode true
!include "MUI2.nsh"
Name "Rift"
OutFile "${OUTPUT}"
InstallDir "$LOCALAPPDATA\Rift"
InstallDirRegKey HKCU "Software\rift\Rift" ""
RequestExecutionLevel user
SetCompressor /SOLID lzma
Icon "${ICON}"
UninstallIcon "${ICON}"
VIProductVersion "${VERSION}.0"
VIAddVersionKey "LegalCopyright" "Rift contributors"
VIAddVersionKey "ProductName" "Rift"
VIAddVersionKey "FileDescription" "Rift installer"
VIAddVersionKey "FileVersion" "${VERSION}"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
Section "Rift"
  SetOutPath "$INSTDIR"
  File /oname=rift.exe "${BINARY}"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  CreateShortcut "$SMPROGRAMS\Rift.lnk" "$INSTDIR\rift.exe"
  WriteRegStr HKCU "Software\rift\Rift" "" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift" "DisplayName" "Rift"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift" "Publisher" "rift"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift" "DisplayIcon" "$INSTDIR\rift.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift" "NoRepair" 1
SectionEnd
Section "Uninstall"
  retry:
  ClearErrors
  Delete "$INSTDIR\rift.exe"
  IfErrors 0 removed
  MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "Close the installed copy of Rift to uninstall it." IDRETRY retry
  Abort
  removed:
  Delete "$SMPROGRAMS\Rift.lnk"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rift"
  DeleteRegKey HKCU "Software\rift\Rift"
SectionEnd
