#!/usr/bin/env bash
# CPU profiling via Linux perf (no code changes). Writes perf.data to tmp/.
# Block profiling would require the binary to expose pprof (e.g. --profile flag).
# Usage: ./profile.sh [workers]   (default workers=24; generates ~100k files)
set -e
NUM_WORKERS=${1:-24}
BIN="./bin/ffd"
PROFILE_OUT_DIR="./tmp"
DATA_DIR="./test/perf/tmp"
FILE_COUNT=100000

if [[ "$(uname)" != Linux ]]; then
  echo "This script uses Linux 'perf' for CPU profiling. On macOS use: go test -cpuprofile=... or Instruments." >&2
  exit 1
fi

mkdir -p "$PROFILE_OUT_DIR"
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"/left "$DATA_DIR"/right

echo "Building binary..."
go build -ldflags="-s -w" -o "$BIN" .

echo "Generating $FILE_COUNT files in $DATA_DIR/left and .../right (all_same)..."
for ((i=0; i<FILE_COUNT; i++)); do
  echo "content $i" > "$DATA_DIR/left/f$i"
  echo "content $i" > "$DATA_DIR/right/f$i"
done

echo "Running ffd under perf (workers=$NUM_WORKERS); output -> $PROFILE_OUT_DIR/perf.data"
perf record -g -o "$PROFILE_OUT_DIR/perf.data" -- "$BIN" --quiet --workers "$NUM_WORKERS" "$DATA_DIR/left" "$DATA_DIR/right" >/dev/null 2>&1

echo ""
echo "Profile written to $PROFILE_OUT_DIR/perf.data"
echo ""
echo "View with perf:"
echo "  perf report -i $PROFILE_OUT_DIR/perf.data"
echo ""
echo "View with Go pprof (flame graph / top):"
echo "  go tool pprof -http=:8080 $BIN $PROFILE_OUT_DIR/perf.data"
echo "  go tool pprof -top $BIN $PROFILE_OUT_DIR/perf.data"
