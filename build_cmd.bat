@echo off
setlocal enabledelayedexpansion

REM One-click build for all cmd targets.
if not exist dist (
    mkdir dist
)

set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

echo Building cmd/crawler...
go build -ldflags="-s -w" -o dist/crawler_linux_amd64 ./cmd/crawler
if not %errorlevel%==0 (
    echo compilation failed: cmd/crawler
    pause
    exit /b 1
)

echo Building cmd/fills...
go build -ldflags="-s -w" -o dist/fills_linux_amd64 ./cmd/fills
if not %errorlevel%==0 (
    echo compilation failed: cmd/fills
    pause
    exit /b 1
)

echo Building cmd/snapshot...
go build -ldflags="-s -w" -o dist/snapshot_linux_amd64 ./cmd/snapshot
if not %errorlevel%==0 (
    echo compilation failed: cmd/snapshot
    pause
    exit /b 1
)

echo compilation succeeded, generated binaries in dist/.
pause
