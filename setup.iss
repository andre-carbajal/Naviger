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
Source: "dist\\naviger-server.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "dist\\naviger-cli.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "dist\\nssm.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "dist\\web_dist\\*"; DestDir: "{app}\\web_dist"; Flags: ignoreversion recursesubdirs

[Icons]
Name: "{group}\\Naviger CLI"; Filename: "{app}\\naviger-cli.exe"

[Registry]
Root: HKLM; Subkey: "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment"; \
    ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; \
    Check: NeedsAddPath('{app}')

[Run]
Filename: "{app}\\nssm.exe"; Parameters: "install NavigerService \"{app}\\naviger-server.exe\""; Flags: runhidden
Filename: "{app}\\nssm.exe"; Parameters: "start NavigerService"; Flags: runhidden

[UninstallRun]
Filename: "{app}\\nssm.exe"; Parameters: "stop NavigerService"; Flags: runhidden
Filename: "{app}\\nssm.exe"; Parameters: "remove NavigerService confirm"; Flags: runhidden

[UninstallDelete]
Type: files; Name: "{app}\\*"; Flags: recursesubdirs
Type: dir; Name: "{app}\\*"
Type: dirifempty; Name: "{app}"

[Code]
function NeedsAddPath(Param: string): boolean;
var
  Path: string;
begin
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, 'SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment', 'Path', Path) then
  begin
    Result := Pos(Uppercase(Param), Uppercase(Path)) = 0;
  end else Result := True;
end;