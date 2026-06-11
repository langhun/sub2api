@echo off
setlocal

set "REPO_ROOT=%~dp0"
set "BASH_EXE="

if exist "C:\Program Files\Git\bin\bash.exe" (
  set "BASH_EXE=C:\Program Files\Git\bin\bash.exe"
)

if not defined BASH_EXE if exist "C:\Program Files\Git\usr\bin\bash.exe" (
  set "BASH_EXE=C:\Program Files\Git\usr\bin\bash.exe"
)

if not defined BASH_EXE (
  for /f "delims=" %%i in ('where bash 2^>nul') do (
    if not defined BASH_EXE set "BASH_EXE=%%i"
  )
)

if not defined BASH_EXE (
  echo Missing bash.exe. Install Git for Windows or add bash.exe to PATH. 1>&2
  exit /b 1
)

pushd "%REPO_ROOT%" >nul || exit /b 1
"%BASH_EXE%" "build-linux.sh"
set "EXIT_CODE=%ERRORLEVEL%"
popd >nul

exit /b %EXIT_CODE%
