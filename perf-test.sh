#!/usr/bin/env bash
# Performance tests for ffd. Builds optimized binary, generates test data under test/perf/tmp/, runs scenarios, appends to perf/perf-results.csv.
# Each CSV record is: date_iso, scenario, file_count, avg_sec_per_pair (average time to compare one file pair).
# Usage: ./perf-test.sh (run from project root)
set -e
BIN="./bin/ffd"
PERF_TMP_DIR="./test/perf/tmp"
PERF_OUTPUT_DIR="./perf"
CSV="${PERF_OUTPUT_DIR}/perf-results.csv"
FILE_COUNTS=(0 1 10 100 1000 10000 100000)

mkdir -p "$PERF_TMP_DIR" "$PERF_OUTPUT_DIR"
echo "Building optimized binary..."
go build -ldflags="-s -w" -o "$BIN" .

if [[ ! -f "$CSV" ]]; then
  echo "date_iso,scenario,file_count,avg_sec_per_pair" > "$CSV"
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
  local left="${PERF_TMP_DIR}/left"
  local right="${PERF_TMP_DIR}/right"
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
  local start end total_sec avg_sec_per_pair
  start=$(date +%s.%N)
  "$BIN" --quiet "$left" "$right" >/dev/null 2>&1
  end=$(date +%s.%N)
  total_sec=$(echo "$end - $start" | bc)
  avg_sec_per_pair="0"
  [[ $n -gt 0 ]] && avg_sec_per_pair=$(echo "scale=6; $total_sec / $n" | bc)
  local date_iso
  date_iso=$(date -Iseconds)
  echo "$date_iso,$scenario,$n,$avg_sec_per_pair" >> "$CSV"
  echo "  $scenario n=$n: total ${total_sec}s, avg ${avg_sec_per_pair}s per pair"
}

date_iso=$(date -Iseconds)
echo "=== Perf run $date_iso ==="
for scenario in all_same left_only right_only; do
  echo "Scenario: $scenario"
  for n in "${FILE_COUNTS[@]}"; do
    run_scenario "$scenario" "$n"
  done
done
echo "Results appended to $CSV (avg sec per pair only)"
