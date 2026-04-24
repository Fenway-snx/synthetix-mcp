@echo off
setlocal enabledelayedexpansion

REM Script to generate gRPC code from proto files for Windows
REM Usage: generate_grpc.bat

echo Generating gRPC code from proto files...

REM Ensure Go bin is in PATH
for /f "tokens=*" %%i in ('go env GOPATH') do set GOPATH=%%i
set PATH=%PATH%;%GOPATH%\bin

REM Check if protoc is installed
where protoc >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: protoc is not installed.
    echo Please install protoc:
    echo   Download from: https://github.com/protocolbuffers/protobuf/releases
    echo   Or use chocolatey: choco install protoc
    echo   Or use winget: winget install Google.Protobuf
    pause
    exit /b 1
)

REM Check if Go plugins are installed
where protoc-gen-go >nul 2>&1
if %errorlevel% neq 0 (
    echo Installing protoc-gen-go...
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
)

where protoc-gen-go-grpc >nul 2>&1
if %errorlevel% neq 0 (
    echo Installing protoc-gen-go-grpc...
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
)

REM Create grpc directory if it doesn't exist
if not exist "grpc" mkdir grpc

REM Generate the code directly in the grpc directory
echo Generating protobuf files...
protoc --go_out=grpc ^
       --go_opt=paths=source_relative ^
       --go-grpc_out=grpc ^
       --go-grpc_opt=paths=source_relative ^
       common.proto matching.proto trading.proto marketdata.proto subaccount.proto

if %errorlevel% neq 0 (
    echo Error: Failed to generate protobuf files
    pause
    exit /b 1
)

REM Fix package names in generated files to use 'grpc' package
echo Fixing package names...
for %%f in (grpc\*.pb.go) do (
    powershell -Command "(Get-Content '%%f') -replace 'package common', 'package grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'package matching', 'package grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'package trading', 'package grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'package marketdata', 'package grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'package subaccount', 'package grpc' | Set-Content '%%f'"
)

REM Fix import paths to use the correct module path
echo Fixing import paths...
for %%f in (grpc\*.pb.go) do (
    powershell -Command "(Get-Content '%%f') -replace 'github.com/Synthetixio/v4-offchain-message/common', 'github.com/Synthetixio/v4-offchain/lib/message/grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'github.com/Synthetixio/v4-offchain-message/matching', 'github.com/Synthetixio/v4-offchain/lib/message/grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'github.com/Synthetixio/v4-offchain-message/trading', 'github.com/Synthetixio/v4-offchain/lib/message/grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'github.com/Synthetixio/v4-offchain-message/marketdata', 'github.com/Synthetixio/v4-offchain/lib/message/grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'github.com/Synthetixio/v4-offchain-message/subaccount', 'github.com/Synthetixio/v4-offchain/lib/message/grpc' | Set-Content '%%f'"
    powershell -Command "(Get-Content '%%f') -replace 'github.com/Synthetixio/v4-offchain-message/grpc', 'github.com/Synthetixio/v4-offchain/lib/message/grpc' | Set-Content '%%f'"
)

echo ✅ gRPC code generation complete!
echo.
echo Generated files:
echo   - grpc/common.pb.go (Common types and enums)
echo   - grpc/matching.pb.go (Matching engine messages)
echo   - grpc/matching_grpc.pb.go (Matching engine services)
echo   - grpc/trading.pb.go (Trading messages)
echo   - grpc/trading_grpc.pb.go (Trading services)
echo   - grpc/marketdata.pb.go (Marketdata messages)
echo   - grpc/marketdata_grpc.pb.go (Marketdata services)
echo   - grpc/subaccount.pb.go (Subaccount messages)
echo   - grpc/subaccount_grpc.pb.go (Subaccount services)
echo.
echo To use in your code:
echo   import "github.com/Synthetixio/v4-offchain/lib/message/grpc"
echo.
echo Example usage:
echo   req := ^&grpc.NewOrderRequest{
echo       Symbol: "BTCUSDT",
echo       Side:   grpc.Side_BUY,
echo       Type:   grpc.OrderType_LIMIT,
echo   }
echo.
pause 
