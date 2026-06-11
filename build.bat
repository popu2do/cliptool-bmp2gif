@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   ClipTool GUI Build
echo ========================================
echo.

echo [1/6] Checking Wails...
where wails >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    set WAILS=wails
    echo Wails found, will use Wails build first.
) else (
    set WAILS=
    echo Wails not found, will use local Go build fallback.
)

echo [2/6] Installing frontend dependencies...
pushd frontend
call npm install
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
popd

echo [3/6] Running frontend tests...
pushd frontend
call npm test
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
popd

echo [4/6] Building frontend assets...
pushd frontend
call npm run build
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
popd

echo [5/6] Running Go tests...
go test ./...
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%

echo [6/6] Building cliptool.exe...
if defined WAILS (
    %WAILS% build -clean -platform windows/amd64
    if %ERRORLEVEL% NEQ 0 (
        echo Wails build failed, falling back to local Go build...
        go build -tags "desktop,production" -ldflags "-H windowsgui" -o build\bin\cliptool.exe .
        if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
    )
) else (
    if not exist build\bin mkdir build\bin
    go build -tags "desktop,production" -ldflags "-H windowsgui" -o build\bin\cliptool.exe .
    if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
)

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
