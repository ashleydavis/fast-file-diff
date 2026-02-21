#!/usr/bin/env bash
# Performance tests for ffd. Builds optimized binary, generates test data, runs scenarios, appends to perf-results.csv.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"
BIN="${SCRIPT_DIR}/bin/ffd"
PERF_DIR="${SCRIPT_DIR}/test/perf"
CSV="${SCRIPT_DIR}/perf-results.csv"
FILE_COUNTS=(0 1 10 100 1000 10000 100000)

mkdir -p "$PERF_DIR"
echo "Building optimized binary..."
go build -ldflags="-s -w" -o "$BIN" .

if [[ ! -f "$CSV" ]]; then
  echo "date_iso,scenario,file_count,total_sec,time_per_file_sec" > "$CSV"
fi

gen_files() {
  local dir="$1"
  local n="$2"
  rm -rf "$dir"
  mkdir -p "$dir"
  for ((i=0; i<n; i++)); do
    echo "content $i" > "$dir/f$i"
  done
}

run_scenario() {
  local scenario="$1"
  local n="$2"
  local left="$PERF_DIR/left"
  local right="$PERF_DIR/right"
  case "$scenario" in
    all_same)
      gen_files "$left" "$n"
      gen_files "$right" "$n"
      for ((i=0; i<n; i++)); do echo "content $i" > "$right/f$i"; done
      ;;
    left_only)
      gen_files "$left" "$n"
      mkdir -p "$right"
      ;;
    right_only)
      mkdir -p "$left"
      gen_files "$right" "$n"
      ;;
    *)
      gen_files "$left" "$n"
      gen_files "$right" "$n"
      ;;
  esac
  local start end total
  start=$(date +%s.%N)
  "$BIN" --quiet "$left" "$right" >/dev/null 2>&1
  end=$(date +%s.%N)
  total=$(echo "$end - $start" | bc)
  local per_file="0"
  [[ $n -gt 0 ]] && per_file=$(echo "scale=6; $total / $n" | bc)
  local date_iso
  date_iso=$(date -Iseconds)
  echo "$date_iso,$scenario,$n,$total,$per_file" >> "$CSV"
  echo "  $scenario n=$n: ${total}s (${per_file}s per file)"
}

date_iso=$(date -Iseconds)
echo "=== Perf run $date_iso ==="
for scenario in all_same left_only right_only; do
  echo "Scenario: $scenario"
  for n in "${FILE_COUNTS[@]}"; do
    run_scenario "$scenario" "$n"
  done
done
echo "Results appended to $CSV"
