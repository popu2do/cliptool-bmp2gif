@echo off
chcp 65001 >nul
echo ╔════════════════════════════════════════╗
echo ║     ClipTool Build Script              ║
echo ╚════════════════════════════════════════╝
echo.

echo [1/3] Cleaning old files...
if exist cliptool.exe del cliptool.exe

echo [2/3] Downloading dependencies...
go mod download

echo [3/3] Building...
go build -ldflags="-s -w" -o cliptool.exe main.go

echo.
if exist cliptool.exe (
    echo [SUCCESS] cliptool.exe
    for %%A in (cliptool.exe) do echo File size: %%~zA bytes
    echo.
    echo Run: cliptool.exe
) else (
    echo [FAILED] Build failed
)

echo.
pause
