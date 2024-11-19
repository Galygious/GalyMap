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

:: Check if GalyMap.log exists, then delete it
echo Checking for existing log.txt file...
if exist GalyMap.log (
    del GalyMap.log
    echo Deleted existing GalyMap.log
) else (
    echo No existing GalyMap.log file found
)

:: Building the main Go program
echo Running the main.exe...
main.exe
echo Main program execution completed.

pause
exit