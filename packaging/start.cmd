@echo off
cd /d "%~dp0"
echo Starting the 3D Job Desk printer bridge. Leave this window open.
echo.
if exist "3d-job-desk-bridge.exe" (
  3d-job-desk-bridge.exe
) else (
  echo Run the downloaded 3d-job-desk-bridge-windows-amd64.exe instead.
)
pause
