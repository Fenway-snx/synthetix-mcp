#! /usr/bin/env bash

# Script to generate gRPC code from proto files
# Usage: ./generate_grpc.sh

set -e

# Ensure Go bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Define here (and here only) the name of the protobuf source files
ProtoSpecs=(
	'trading::Trading'
	'common::Common'
	'marketconfig::Market Config'
	'marketdata::Market Data'
	'matching::Matching'
	'subaccount::SubAccount'
	'websocket::WebSocket'
	'pricing::Pricing'
)
ProtoNames=()
ProtoFileNames=()

for protospec in "${ProtoSpecs[@]}" ; do
	key="${protospec%%::*}"
	value="${protospec##*::}"

	ProtoNames+=("$key")
	ProtoFileNames+=("$key.proto")
done

echo "Generating gRPC code from proto files..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
	echo "Error: protoc is not installed."
	echo "Please install protoc:"
	echo "  macOS: brew install protobuf"
	echo "  Linux: apt-get install protobuf-compiler"
	exit 1
fi

# Check if Go plugins are installed
if ! command -v protoc-gen-go &> /dev/null; then
	echo "Installing protoc-gen-go..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
	echo "Installing protoc-gen-go-grpc..."
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Create grpc directory if it doesn't exist
mkdir -p grpc

# Generate the code directly in the grpc directory
echo "Generating protobuf files..."
protoc --go_out=grpc \
	--go_opt=paths=source_relative \
	--go-grpc_out=grpc \
	--go-grpc_opt=paths=source_relative \
	${ProtoFileNames[*]}

# Fix package names in generated files to use 'grpc' package
echo "Fixing package names..."

## TODO: convert to use lists (if useful at all)

sed -i '' 's/package trading/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package trading/package grpc/g' grpc/*.pb.go

sed -i '' 's/package common/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package common/package grpc/g' grpc/*.pb.go

sed -i '' 's/package marketconfig/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package marketconfig/package grpc/g' grpc/*.pb.go

sed -i '' 's/package marketdata/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package marketdata/package grpc/g' grpc/*.pb.go

sed -i '' 's/package matching/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package matching/package grpc/g' grpc/*.pb.go

sed -i '' 's/package subaccount/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package subaccount/package grpc/g' grpc/*.pb.go

sed -i '' 's/package websocket/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package websocket/package grpc/g' grpc/*.pb.go

sed -i '' 's/package pricing/package grpc/g' grpc/*.pb.go 2>/dev/null || \
sed -i 's/package pricing/package grpc/g' grpc/*.pb.go

# Fix import paths to use the correct module path
echo "Fixing import paths..."

## TODO: convert to use lists (if useful at all)

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/trading|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/trading|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/common|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/common|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/marketconfig|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/marketconfig|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/marketdata|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/marketdata|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/matching|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/matching|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/subaccount|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/subaccount|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/websocket|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/websocket|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

sed -i '' 's|github.com/Synthetixio/v4-offchain-message/pricing|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/pricing|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

# Fix any remaining references in protobuf descriptor strings
sed -i '' 's|github.com/Synthetixio/v4-offchain-message/grpc|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go 2>/dev/null || \
sed -i 's|github.com/Synthetixio/v4-offchain-message/grpc|github.com/Synthetixio/v4-offchain/lib/message/grpc|g' grpc/*.pb.go

echo "✅ gRPC code generation complete!"
echo ""
echo "Generated files:"
for protospec in "${ProtoSpecs[@]}" ; do
	key="${protospec%%::*}"
	value="${protospec##*::}"

	echo "  - grpc/${key}_grpc.pb.go ($value services)"
	echo "  - grpc/${key}.pb.go ($value messages)"
done
echo ""
echo "To use in your code:"
echo '  import "github.com/Synthetixio/v4-offchain/lib/message/grpc"'
echo ""
echo "Example usage:"
echo '  req := &grpc.NewOrderRequest{'
echo '      Symbol: "BTCUSDT",'
echo '      Side:   grpc.Side_BUY,'
echo '      Type:   grpc.OrderType_LIMIT,'
echo '  }'
