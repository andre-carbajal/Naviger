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
call go build -v -o dist\mc-manager-server.exe .\cmd\server
echo "Building CLI..."
call go build -v -o dist\mc-cli.exe .\cmd\cli

echo "Build finished successfully!"
endlocal
