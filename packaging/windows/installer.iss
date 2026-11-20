; Inno Setup script for PC Monitor (per-user install, no admin needed).
; Built by the release workflow:
;   set VERSION=1.6.0 & set ARCH=amd64 & set BINDIR=...\build\windows_amd64
;   ISCC.exe packaging\windows\installer.iss
; Self-update works afterwards because the install folder is user-writable.

#define AppVersion GetEnv("VERSION")
#define Arch GetEnv("ARCH")
#define BinDir GetEnv("BINDIR")
#define OutDir GetEnv("OUTDIR")

[Setup]
AppId={{8F2C3B5A-6E1D-4C7A-9B2E-3D4F5A6B7C8D}
AppName=PC Monitor
AppVersion={#AppVersion}
AppVerName=PC Monitor {#AppVersion}
AppPublisher=garrettcorn
AppPublisherURL=https://github.com/garrettcorn/remote-monitor
AppSupportURL=https://github.com/garrettcorn/remote-monitor/issues
AppUpdatesURL=https://github.com/garrettcorn/remote-monitor/releases
DefaultDirName={localappdata}\Programs\PC Monitor
DefaultGroupName=PC Monitor
DisableProgramGroupPage=yes
DisableDirPage=auto
PrivilegesRequired=lowest
OutputDir={#OutDir}
OutputBaseFilename=pc_monitor_{#AppVersion}_windows_{#Arch}_setup
SetupIconFile=pc-monitor.ico
UninstallDisplayIcon={app}\pc_monitor-app.exe
UninstallDisplayName=PC Monitor
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
#if Arch == "arm64"
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
#else
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
#endif

[Tasks]
Name: "autostart"; Description: "Start PC Monitor when I log in"; Flags: checkedonce
Name: "desktopicon"; Description: "Create a desktop shortcut"; Flags: unchecked

[Files]
Source: "{#BinDir}\pc_monitor.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BinDir}\pc_monitor-app.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\CHANGELOG.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\PC Monitor"; Filename: "{app}\pc_monitor-app.exe"; Comment: "Show the PC Monitor tray icon"
Name: "{group}\PC Monitor Settings"; Filename: "{app}\pc_monitor-app.exe"; Parameters: "settings"; Comment: "Pairing QR code, PIN and options"
Name: "{group}\Uninstall PC Monitor"; Filename: "{uninstallexe}"
Name: "{autodesktop}\PC Monitor"; Filename: "{app}\pc_monitor-app.exe"; Tasks: desktopicon

[Registry]
; Same per-user Run entry the app manages itself ("Start at login").
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "PC Monitor"; ValueData: """{app}\pc_monitor-app.exe"""; Flags: uninsdeletevalue; Tasks: autostart
; Also remove it on uninstall if it was turned on later from the app.
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: none; ValueName: "PC Monitor"; Flags: uninsdeletevalue

[Run]
Filename: "{app}\pc_monitor-app.exe"; Description: "Start PC Monitor now"; Flags: nowait postinstall skipifsilent
Filename: "{app}\pc_monitor-app.exe"; Parameters: "settings"; Description: "Open settings (pairing QR code and PIN)"; Flags: nowait postinstall skipifsilent unchecked

[UninstallRun]
Filename: "{app}\pc_monitor.exe"; Parameters: "stop"; Flags: runhidden; RunOnceId: "StopServer"
Filename: "{cmd}"; Parameters: "/C taskkill /IM pc_monitor-app.exe /F"; Flags: runhidden; RunOnceId: "StopTray"
