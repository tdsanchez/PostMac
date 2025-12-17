#!/bin/bash

set -e

PORT="${1:-8080}"
DIR="${2:-.}"

echo "PWD: $PWD"

echo "=== Go Media Server Builder (with Tag Categories) ==="
echo "Port: $PORT"
echo "Directory: $DIR"
echo ""

# Kill any existing server
pkill -9 media-server 2>/dev/null || true

echo "Building server..."
go build -o media-server ./cmd/media-server

echo "✅ Build complete!"
echo ""

./media-server -port "$PORT" -dir "$DIR" &
SERVER_PID=$!

# Wait for server to be fully ready by checking if it responds
URL="http://localhost:$PORT"
echo "⏳ Waiting for server to be ready..."
for i in {1..30}; do
    if curl -s -o /dev/null -w "%{http_code}" "$URL" | grep -q "200"; then
        echo "✅ Server is ready!"
        break
    fi
    sleep 0.5
done

echo "🌐 Opening $URL in browser..."
open "$URL"

wait $SERVER_PID
