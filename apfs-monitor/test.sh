#!/bin/bash

# Quick test script to verify APFS monitor functionality

echo "=== APFS Monitor Test ==="
echo ""

echo "1. Checking if Go is installed..."
if ! command -v go &> /dev/null; then
    echo "   ERROR: Go is not installed"
    echo "   Install from: https://golang.org/dl/"
    exit 1
fi
echo "   ✓ Go version: $(go version)"
echo ""

echo "2. Building APFS monitor..."
if go build -o apfs-monitor main.go; then
    echo "   ✓ Build successful"
else
    echo "   ERROR: Build failed"
    exit 1
fi
echo ""

echo "3. Testing diskutil access..."
if diskutil apfs list &> /dev/null; then
    echo "   ✓ diskutil accessible"
else
    echo "   ERROR: Cannot run diskutil"
    exit 1
fi
echo ""

echo "4. Running one-time check..."
./apfs-monitor
echo ""

echo "5. Current APFS container status:"
diskutil apfs list | grep -E "(APFS Container|Capacity Free|Capacity Consumed)" | head -20
echo ""

echo "=== Test Complete ==="
echo ""
echo "If you see space warnings above, consider:"
echo "  - Increasing threshold values"
echo "  - Freeing up space on your container"
echo "  - Removing unnecessary volumes/partitions"
echo ""
echo "To install as daemon: ./install.sh"
