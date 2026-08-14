Unicode true

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"

!ifndef VERSION
  !define VERSION "1.0.0"
!endif
!ifndef FILE_VERSION
  !define FILE_VERSION "1.0.0.0"
!endif
!ifndef APP_EXE
  !error "APP_EXE must point to RegionLockpaw.exe"
!endif
!ifndef OUTPUT_EXE
  !define OUTPUT_EXE "RegionLockpawSetup.exe"
!endif

Name "Region Lockpaw"
OutFile "${OUTPUT_EXE}"
InstallDir "$LOCALAPPDATA\Programs\Region Lockpaw"
InstallDirRegKey HKCU "Software\RegionLockpaw" "InstallLocation"
RequestExecutionLevel user
SetCompressor /SOLID lzma
SetCompressorDictSize 32
ShowInstDetails show
ShowUninstDetails show

VIProductVersion "${FILE_VERSION}"
VIAddVersionKey /LANG=1033 "ProductName" "Region Lockpaw"
VIAddVersionKey /LANG=1033 "FileDescription" "Region Lockpaw Setup"
VIAddVersionKey /LANG=1033 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1033 "ProductVersion" "${VERSION}"
VIAddVersionKey /LANG=1033 "LegalCopyright" "Copyright (c) 2026 Region Lockpaw contributors"

!define MUI_ABORTWARNING
!define MUI_ICON "..\assets\RegionLockpaw.ico"
!define MUI_UNICON "..\assets\RegionLockpaw.ico"
!define MUI_FINISHPAGE_RUN "$INSTDIR\RegionLockpaw.exe"
!define MUI_FINISHPAGE_RUN_TEXT "$(LaunchApp)"
!define MUI_LANGDLL_REGISTRY_ROOT HKCU
!define MUI_LANGDLL_REGISTRY_KEY "Software\RegionLockpaw"
!define MUI_LANGDLL_REGISTRY_VALUENAME "InstallerLanguage"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

LangString LaunchApp ${LANG_SIMPCHINESE} "启动 Region Lockpaw"
LangString LaunchApp ${LANG_ENGLISH} "Launch Region Lockpaw"
LangString AlreadyInstalled ${LANG_SIMPCHINESE} "将更新现有的 Region Lockpaw 安装。"
LangString AlreadyInstalled ${LANG_ENGLISH} "The existing Region Lockpaw installation will be updated."

Var IsUpgrade

Function .onInit
  !insertmacro MUI_LANGDLL_DISPLAY
  SetShellVarContext current
  StrCpy $IsUpgrade "0"

  ReadRegStr $0 HKCU "Software\RegionLockpaw" "InstallLocation"
  ${If} $0 != ""
    StrCpy $IsUpgrade "1"
    MessageBox MB_OK|MB_ICONINFORMATION "$(AlreadyInstalled)"
  ${EndIf}
FunctionEnd

Function un.onInit
  !insertmacro MUI_UNGETLANGUAGE
  SetShellVarContext current
FunctionEnd

Function CloseRunningApp
  FindWindow $0 "RegionLockpaw.Controller" "Region Lockpaw Controller"
  ${If} $0 != 0
    SendMessage $0 ${WM_CLOSE} 0 0 /TIMEOUT=3000
    Sleep 1000
  ${EndIf}
FunctionEnd

Section "Region Lockpaw" SectionMain
  SectionIn RO
  Call CloseRunningApp

  SetOutPath "$INSTDIR"
  File /oname=RegionLockpaw.exe "${APP_EXE}"
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  CreateDirectory "$SMPROGRAMS\Region Lockpaw"
  CreateShortcut "$SMPROGRAMS\Region Lockpaw\Region Lockpaw.lnk" "$INSTDIR\RegionLockpaw.exe"
  CreateShortcut "$SMPROGRAMS\Region Lockpaw\Uninstall Region Lockpaw.lnk" "$INSTDIR\Uninstall.exe"

  ${If} $IsUpgrade != "1"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "RegionLockpaw" "$\"$INSTDIR\RegionLockpaw.exe$\" --background"
  ${EndIf}
  WriteRegStr HKCU "Software\RegionLockpaw" "InstallLocation" "$INSTDIR"

  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "DisplayName" "Region Lockpaw"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "Publisher" "Region Lockpaw Project"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "URLInfoAbout" "https://github.com/william08190/region-lockpaw"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "DisplayIcon" "$INSTDIR\RegionLockpaw.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "QuietUninstallString" "$\"$INSTDIR\Uninstall.exe$\" /S"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "NoRepair" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw" "EstimatedSize" 5120
SectionEnd

Section "Uninstall"
  Call un.CloseRunningApp

  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "RegionLockpaw"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RegionLockpaw"
  DeleteRegKey HKCU "Software\RegionLockpaw"

  Delete "$SMPROGRAMS\Region Lockpaw\Region Lockpaw.lnk"
  Delete "$SMPROGRAMS\Region Lockpaw\Uninstall Region Lockpaw.lnk"
  RMDir "$SMPROGRAMS\Region Lockpaw"

  Delete "$INSTDIR\RegionLockpaw.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  RMDir /r "$LOCALAPPDATA\Region Lockpaw"
SectionEnd

Function un.CloseRunningApp
  FindWindow $0 "RegionLockpaw.Controller" "Region Lockpaw Controller"
  ${If} $0 != 0
    SendMessage $0 ${WM_CLOSE} 0 0 /TIMEOUT=3000
    Sleep 1000
  ${EndIf}
FunctionEnd
