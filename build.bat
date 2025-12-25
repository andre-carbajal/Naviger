@echo off
setlocal

echo "Cleaning up previous build..."
if exist dist (
    rmdir /s /q dist
)

echo "Building web frontend..."
pushd web
call npm install
call npm run build
popd

mkdir dist\web_dist
xcopy /s /e /i /y web\dist\* dist\web_dist\

echo "Building Go backend..."
echo "Building server..."
call go build -v -o dist\naviger-server.exe .\cmd\server
echo "Building CLI..."
call go build -v -o dist\naviger-cli.exe .\cmd\cli

echo "Downloading NSSM..."
powershell -Command "Invoke-WebRequest -Uri https://nssm.cc/release/nssm-2.24.zip -OutFile nssm.zip"
powershell -Command "Expand-Archive -Path nssm.zip -DestinationPath nssm_temp"
copy nssm_temp\nssm-2.24\win64\nssm.exe dist\
del nssm.zip
rmdir /s /q nssm_temp

echo "Build finished successfully!"
endlocal
