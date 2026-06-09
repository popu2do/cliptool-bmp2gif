@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   ClipTool GUI Build
echo ========================================
echo.

echo [1/5] Checking Wails...
where wails >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    set WAILS=wails
) else (
    set WAILS=go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
)

echo [2/5] Installing frontend dependencies...
pushd frontend
call npm install
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
popd

echo [3/5] Running frontend tests...
pushd frontend
call npm test
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
popd

echo [4/5] Running Go tests...
go test ./...
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%

echo [5/5] Building cliptool.exe...
%WAILS% build -clean -platform windows/amd64
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%

if exist build\bin\cliptool.exe (
    copy /Y build\bin\cliptool.exe cliptool.exe >nul
    copy /Y build\bin\cliptool.exe ..\..\cliptool.exe >nul
    echo.
    echo [SUCCESS] cliptool.exe
    for %%A in (cliptool.exe) do echo File size: %%~zA bytes
) else (
    echo [FAILED] build\bin\cliptool.exe not found
    exit /b 1
)

echo.
pause
