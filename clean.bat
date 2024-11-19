@echo off
:: Check for administrator privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    :: Re-run the batch file as administrator
    echo Requesting administrator privileges...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

:: Log starting point
echo Starting batch process...

:: Navigate to the directory where this batch file is located
echo Navigating to the batch file directory...
cd /d "%~dp0"

:: Check if log.txt exists, then delete it
echo Checking for existing log.txt file...
if exist log.txt (
    del log.txt
    echo Deleted existing log.txt
) else (
    echo No existing log.txt file found
)

echo Removing vendor
rmdir /s /q vendor

echo Cleaning go cache
go clean -modcache

echo removing go sum
del go.sum

echo Clean completed. All go files ready for reimplementation.
pause
exit

