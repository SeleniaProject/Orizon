#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."

ART=artifacts
mkdir -p "$ART"

echo "==> Build CLIs"
go build -o orizon-test ./cmd/orizon-test
go build -o orizon-fuzz ./cmd/orizon-fuzz
go build -o orizon-repro ./cmd/orizon-repro

echo "==> Unit tests (short, retries, junit/json)"
./orizon-test --packages ./... --short --retries 1 --fail-fast --junit "$ART/junit.xml" --json-summary "$ART/junit_summary.json" --color=false || true

echo "==> Go test coverage"
go test -coverprofile "$ART/cover.out" ./... || true
go tool cover -func "$ART/cover.out" > "$ART/cover.txt" || true

echo "==> Fuzz smokes"
PARSER_CORPUS="corpus/parser_corpus.txt"
LEXER_CORPUS="corpus/lexer_corpus.txt"
AB_CORPUS="corpus/astbridge_corpus.txt"

CMD_PARSER=(./orizon-fuzz --target parser --duration 2s --p 2 --max-execs 4000 --cov-mode trigram --stats --json-stats "$ART/stats_parser.json" --save-seed "$ART/seed.txt" --out "$ART/crashes.txt" --min-on-crash --min-dir "$ART/crashes_min" --min-budget 2s)
if [[ -f "$PARSER_CORPUS" ]]; then CMD_PARSER+=(--corpus "$PARSER_CORPUS" --covout "$ART/parser.cov" --covstats); fi
"${CMD_PARSER[@]}" || true

CMD_PARSER_LAX=(./orizon-fuzz --target parser-lax --duration 2s --p 2 --max-execs 3000 --stats --json-stats "$ART/stats_parser_lax.json")
if [[ -f "$PARSER_CORPUS" ]]; then CMD_PARSER_LAX+=(--corpus "$PARSER_CORPUS"); fi
"${CMD_PARSER_LAX[@]}" || true

CMD_LEXER=(./orizon-fuzz --target lexer --duration 2s --p 2 --max-execs 3000 --stats)
if [[ -f "$LEXER_CORPUS" ]]; then CMD_LEXER+=(--corpus "$LEXER_CORPUS"); fi
"${CMD_LEXER[@]}" || true

CMD_AB=(./orizon-fuzz --target astbridge --duration 2s --p 2 --max-execs 3000 --stats)
if [[ -f "$AB_CORPUS" ]]; then CMD_AB+=(--corpus "$AB_CORPUS"); fi
"${CMD_AB[@]}" || true

CMD_HIR=(./orizon-fuzz --target hir --duration 2s --p 2 --max-execs 3000 --stats)
if [[ -f "$PARSER_CORPUS" ]]; then CMD_HIR+=(--corpus "$PARSER_CORPUS"); fi
"${CMD_HIR[@]}" || true

CMD_AB_HIR=(./orizon-fuzz --target astbridge-hir --duration 2s --p 2 --max-execs 3000 --stats)
if [[ -f "$AB_CORPUS" ]]; then CMD_AB_HIR+=(--corpus "$AB_CORPUS"); fi
"${CMD_AB_HIR[@]}" || true

echo "==> Reproduce/minimize last crash (if any)"
if [[ -s "$ART/crashes.txt" ]]; then
  ./orizon-repro --log "$ART/crashes.txt" --out "$ART/minimized.bin" --budget 2s --target parser || true
fi

echo "==> Compose summary"
go build -o orizon-summary ./cmd/orizon-summary
./orizon-summary --junit-summary "$ART/junit_summary.json" --stats "$ART/stats_parser.json,$ART/stats_parser_lax.json" --cover "$ART/cover.txt" --out "$ART/summary.md" | tee -a "$GITHUB_STEP_SUMMARY" || true
if [[ -f "$ART/cover.txt" ]]; then
  echo "\n#### Coverage" | tee -a "$GITHUB_STEP_SUMMARY"
  tail -n 1 "$ART/cover.txt" | sed -e 's/^/- /' | tee -a "$GITHUB_STEP_SUMMARY"
  echo "" | tee -a "$GITHUB_STEP_SUMMARY"
fi

echo
echo "All smoke steps completed. Artifacts in '$ART'"


