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

:: Update dependencies, create vendor directory, and tidy up
echo Updating dependencies with 'go get -u'...
go get -u
echo Dependencies updated.

echo Creating vendor directory with 'go mod vendor'...
go mod vendor
echo Vendor directory created.

echo Tidying up modules with 'go mod tidy'...
go mod tidy
echo Modules tidied up.

:: Building the main Go program
echo Building the main Go program...
go build -x -o main.exe main.go
echo Main program built.

pause
exit
