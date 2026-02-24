#!/usr/bin/env bash
# Performance tests for ffd. Builds optimized binary, generates test data under test/perf/tmp/, runs scenarios, appends one row per run to perf/perf-results.csv.
# Each CSV row: date_iso, machine, workers, min_sec_per_pair, max_sec_per_pair, avg_sec_per_pair, longest_test_total_sec, longest_test (scenario_filecount of the test that took the longest).
# machine describes the host (OS, arch, cores, CPU model) so results can be compared across runs on different hardware.
# Usage: ./perf-test.sh (run from project root)
set -e
NUM_WORKERS=24
BIN="./bin/ffd"
PERF_TMP_DIR="./test/perf/tmp"
PERF_OUTPUT_DIR="./perf"
CSV="${PERF_OUTPUT_DIR}/perf-results.csv"
RESULTS_TMP=$(mktemp)
trap 'rm -f "$RESULTS_TMP"' EXIT
FILE_COUNTS=(0 1 10 100 1000 10000 100000)

mkdir -p "$PERF_OUTPUT_DIR"
rm -rf "$PERF_TMP_DIR"
mkdir -p "$PERF_TMP_DIR"
echo "Building optimized binary..."
go build -ldflags="-s -w" -o "$BIN" .

# Capture machine spec once (Linux/macOS). Safe for CSV: no newlines; commas in CPU model replaced with semicolons.
get_machine_spec() {
  local os arch cores cpu
  os=$(uname -s 2>/dev/null || echo "unknown")
  arch=$(uname -m 2>/dev/null || echo "unknown")
  if [[ -f /proc/cpuinfo ]]; then
    cores=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || echo "?")
    cpu=$(grep -m1 "model name" /proc/cpuinfo 2>/dev/null | sed 's/^[^:]*: *//; s/,/;/g' || echo "?")
  else
    cores=$(sysctl -n hw.ncpu 2>/dev/null || echo "?")
    cpu=$(sysctl -n machdep.cpu.brand_string 2>/dev/null | sed 's/,/;/g' || echo "?")
  fi
  echo "${os} ${arch} ${cores}cores ${cpu}"
}

MACHINE_SPEC=$(get_machine_spec)
echo "Machine: $MACHINE_SPEC"

if [[ ! -f "$CSV" ]]; then
  echo "date_iso,machine,workers,min_sec_per_pair,max_sec_per_pair,avg_sec_per_pair,longest_test_total_sec,longest_test" > "$CSV"
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

# Runs one scenario, prints human-readable line to stderr, appends "scenario,file_count,total_sec,avg_sec_per_pair" to RESULTS_TMP.
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
    all_different)
      gen_files "$left" "$n"
      rm -rf "$right"
      mkdir -p "$right"
      for ((i=0; i<n; i++)); do echo "other $i" > "$right/f$i"; done
      ;;
    left_only)
      gen_files "$left" "$n"
      rm -rf "$right"
      mkdir -p "$right"
      ;;
    right_only)
      rm -rf "$left"
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
  "$BIN" --quiet --workers "$NUM_WORKERS" "$left" "$right" >/dev/null 2>&1
  end=$(date +%s.%N)
  total_sec=$(echo "$end - $start" | bc)
  avg_sec_per_pair="0"
  [[ $n -gt 0 ]] && avg_sec_per_pair=$(echo "scale=6; $total_sec / $n" | bc)
  echo "$scenario,$n,$total_sec,$avg_sec_per_pair" >> "$RESULTS_TMP"
  echo "  $scenario n=$n: total ${total_sec}s, avg ${avg_sec_per_pair}s per pair"
}

date_iso=$(date -Iseconds)
echo "=== Perf run $date_iso ==="
: > "$RESULTS_TMP"
for scenario in all_same all_different left_only right_only; do
  echo "Scenario: $scenario"
  for n in "${FILE_COUNTS[@]}"; do
    run_scenario "$scenario" "$n"
  done
done

# One row per run: min/max/avg sec per pair, longest test total sec, and which test was longest (scenario_filecount).
stats=$(awk -F, '
BEGIN { min_avg=1e99; max_avg=-1; sum_avg=0; count=0; max_total=-1; longest="" }
{
  scenario=$1; n=$2; total=$3+0; avg=$4+0
  if (avg < min_avg) min_avg = avg
  if (avg > max_avg) max_avg = avg
  sum_avg += avg; count++
  if (total > max_total) { max_total = total; longest = scenario "_" n }
}
END {
  avg_sec = (count > 0) ? sum_avg/count : 0
  print min_avg "," max_avg "," avg_sec "," max_total "," longest
}' "$RESULTS_TMP")
qm="${MACHINE_SPEC//\"/\"\"}"
echo "$date_iso,\"$qm\",$NUM_WORKERS,$stats" >> "$CSV"
echo "Results appended to $CSV (one row: min/max/avg sec per pair, longest test total, longest test)"
