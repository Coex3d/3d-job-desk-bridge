@echo off
cd /d "%~dp0"
echo Pair this computer with your 3D Job Desk.
echo Get a pairing code from Printers on the website first.
echo.
if exist "3d-job-desk-bridge.exe" (
  3d-job-desk-bridge.exe pair
) else (
  echo Run the downloaded 3d-job-desk-bridge-windows-amd64.exe instead.
)
echo.
pause
