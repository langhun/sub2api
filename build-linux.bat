@echo off
setlocal enabledelayedexpansion

echo Building Sub2API Linux amd64 production binary...

set "VERSION_FILE=backend\cmd\server\VERSION"
set "CONFIG_EXAMPLE=deploy\config.example.yaml"
set "UPLOAD_DIR=dist\upload"
set "BINARY_NAME=sub2api-linux-amd64"
set "PACKAGE_BASENAME=%BINARY_NAME%"

if not exist "%VERSION_FILE%" (
  echo Missing version file: %VERSION_FILE% 1>&2
  exit /b 1
)

if not exist "%CONFIG_EXAMPLE%" (
  echo Missing config example: %CONFIG_EXAMPLE% 1>&2
  exit /b 1
)

set /p VERSION=<%VERSION_FILE%
for /f "tokens=*" %%i in ('git rev-parse --short HEAD') do set COMMIT=%%i
for /f "tokens=*" %%i in ('powershell -NoProfile -Command "(Get-Date).ToUniversalTime().ToString(''yyyy-MM-ddTHH:mm:ssZ'')"') do set DATE=%%i
set "BUILD_TYPE=production"

echo Version: %VERSION%
echo Commit: %COMMIT%
echo Date: %DATE%
echo BuildType: %BUILD_TYPE%

echo Cleaning old upload artifacts...
if exist "%UPLOAD_DIR%" rmdir /s /q "%UPLOAD_DIR%"
mkdir "%UPLOAD_DIR%"

echo Building frontend into backend\internal\web\dist...
call :build_frontend
if errorlevel 1 exit /b 1

echo Building backend for linux/amd64...
pushd backend
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -tags embed -ldflags "-s -w -X 'main.Version=%VERSION%' -X 'main.Commit=%COMMIT%' -X 'main.Date=%DATE%' -X 'main.BuildType=%BUILD_TYPE%'" -trimpath -o "..\%UPLOAD_DIR%\%BINARY_NAME%" .\cmd\server
if errorlevel 1 exit /b 1
popd

echo Copying release metadata and example config...
copy "%VERSION_FILE%" "%UPLOAD_DIR%\VERSION" >nul
copy "%CONFIG_EXAMPLE%" "%UPLOAD_DIR%\config.example.yaml" >nul

echo Creating release package...
pushd "%UPLOAD_DIR%"
where zstd >nul 2>nul
if errorlevel 1 (
  set "PACKAGE=%PACKAGE_BASENAME%.tar.gz"
  tar -czf !PACKAGE! %BINARY_NAME% VERSION config.example.yaml
) else (
  set "PACKAGE=%PACKAGE_BASENAME%.tar.zst"
  tar -cf %PACKAGE_BASENAME%.tar %BINARY_NAME% VERSION config.example.yaml
  if errorlevel 1 exit /b 1
  zstd -19 %PACKAGE_BASENAME%.tar -o !PACKAGE!
  if errorlevel 1 exit /b 1
  del %PACKAGE_BASENAME%.tar
)
if errorlevel 1 exit /b 1
certutil -hashfile %BINARY_NAME% SHA256 | findstr /v ":" > %BINARY_NAME%.sha256
certutil -hashfile !PACKAGE! SHA256 | findstr /v ":" > !PACKAGE!.sha256
popd

echo Build complete. Artifacts:
dir /b "%UPLOAD_DIR%"
echo SHA256:
type "%UPLOAD_DIR%\%BINARY_NAME%.sha256"
type "%UPLOAD_DIR%\!PACKAGE!.sha256"

endlocal
exit /b 0

:build_frontend
where pnpm >nul 2>nul
if not errorlevel 1 (
  pnpm --dir frontend run build
  exit /b !errorlevel!
)

where corepack >nul 2>nul
if not errorlevel 1 (
  corepack pnpm --dir frontend run build
  exit /b !errorlevel!
)

if exist "frontend\node_modules\.bin\vite.cmd" (
  call "frontend\node_modules\.bin\vite.cmd" build
  exit /b !errorlevel!
)

echo Missing pnpm/corepack, and frontend\node_modules\.bin\vite.cmd was not found. 1>&2
exit /b 1
