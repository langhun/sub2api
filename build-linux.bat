@echo off
setlocal enabledelayedexpansion

echo 🚀 开始构建 Sub2API Linux amd64 版本...

REM 读取版本信息
set /p VERSION=<VERSION
for /f "tokens=*" %%i in ('git rev-parse --short HEAD') do set COMMIT=%%i
for /f "tokens=1-3 delims=/ " %%a in ('date /t') do set DATE=%%c-%%a-%%b
for /f "tokens=1-2 delims=: " %%a in ('time /t') do set TIME=%%a:%%b
set BUILD_TYPE=production

echo 📦 版本信息:
echo   Version: %VERSION%
echo   Commit: %COMMIT%
echo   Date: %DATE%T%TIME%:00Z
echo   BuildType: %BUILD_TYPE%

REM 清理旧的构建产物
echo 🧹 清理旧的构建产物...
if exist dist\upload rmdir /s /q dist\upload
mkdir dist\upload

REM 等待前端构建完成
echo ⏳ 等待前端构建完成...
:wait_frontend
if not exist "frontend\dist" (
    timeout /t 2 /nobreak >nul
    goto wait_frontend
)
echo ✅ 前端构建完成

REM 复制前端构建产物到后端
echo 📋 复制前端构建产物...
if exist backend\internal\web\dist rmdir /s /q backend\internal\web\dist
xcopy /E /I /Y frontend\dist backend\internal\web\dist >nul

REM 构建后端 (Linux amd64)
echo 🔨 构建后端 (Linux amd64)...
cd backend

set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

go build -tags embed -ldflags "-s -w -X 'main.Version=%VERSION%' -X 'main.Commit=%COMMIT%' -X 'main.Date=%DATE%T%TIME%:00Z' -X 'main.BuildType=%BUILD_TYPE%'" -o ..\dist\upload\sub2api-linux-amd64 .\cmd\server

cd ..

REM 复制配置文件和 VERSION
echo 📄 复制配置文件...
copy VERSION dist\upload\ >nul
copy config.example.yaml dist\upload\ >nul

REM 创建 tar.zst 压缩包 (使用 7z 和 zstd)
echo 📦 创建 tar.zst 压缩包...
cd dist\upload
tar -cf sub2api-%VERSION%-linux-amd64.tar sub2api-linux-amd64 VERSION config.example.yaml
zstd -19 sub2api-%VERSION%-linux-amd64.tar -o sub2api-%VERSION%-linux-amd64.tar.zst
del sub2api-%VERSION%-linux-amd64.tar
cd ..\..

REM 计算 hash
echo 🔐 计算文件 hash...
cd dist\upload
certutil -hashfile sub2api-linux-amd64 SHA256 | findstr /v ":" > sub2api-linux-amd64.sha256
certutil -hashfile sub2api-%VERSION%-linux-amd64.tar.zst SHA256 | findstr /v ":" > sub2api-%VERSION%-linux-amd64.tar.zst.sha256
cd ..\..

REM 显示构建结果
echo.
echo ✅ 构建完成！
echo.
echo 📦 构建产物:
dir /b dist\upload
echo.
echo 🔐 SHA256:
type dist\upload\sub2api-linux-amd64.sha256
type dist\upload\sub2api-%VERSION%-linux-amd64.tar.zst.sha256
echo.
echo 📤 上传文件: dist\upload\sub2api-%VERSION%-linux-amd64.tar.zst

endlocal
