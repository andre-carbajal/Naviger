#define MyAppName "Naviger"
#define MyAppVersion GetEnv("MYAPP_VERSION")
#define MyAppPublisher "Andre Carbajal"

[Setup]
AppId={{628b9b2c-84a9-4010-9a9c-10f3b32b538c}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
OutputBaseFilename=NavigerSetup
Compression=lzma
SolidCompression=yes
PrivilegesRequired=admin

[Files]
Source: "dist\naviger-server.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "dist\naviger-cli.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "dist\web_dist\*"; DestDir: "{app}\web_dist"; Flags: ignoreversion recursesubdirs

[Icons]
Name: "{group}\{#MyAppName} Server"; Filename: "{app}\naviger-server.exe"
Name: "{group}\Naviger Web UI"; Filename: "http://localhost:23008"

[Registry]
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Check: NeedsAddPath('{app}')

[Run]
Filename: "{app}\naviger-server.exe"; Description: "Iniciar el servidor de Naviger"; Flags: postinstall nowait skipifsilent
Filename: "http://localhost:23008"; Description: "Abrir la interfaz web (localhost:23008)"; Flags: postinstall shellexec skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{app}\web_dist"
Type: files; Name: "{app}\*"
Type: dirifempty; Name: "{app}"

[Code]
function NeedsAddPath(Param: string): boolean;
var
  Path: string;
begin
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment', 'Path', Path) then
  begin
    Result := Pos(Uppercase(Param), Uppercase(Path)) = 0;
  end else Result := True;
end;