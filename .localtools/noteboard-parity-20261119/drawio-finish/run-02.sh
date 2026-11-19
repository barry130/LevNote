#!/usr/bin/env bash
set -uo pipefail
cd /f/AIproject/LevNote || exit 2
evidence=.localtools/noteboard-parity-20261119/drawio-finish
printf "%s\n" "bash .localtools/gate.sh test -count=1 -timeout=300s ./modules/graphview -run Drawio.*Clone -v" > "$evidence/02-command.txt"
bash .localtools/gate.sh test -count=1 -timeout=300s ./modules/graphview -run "Drawio.*Clone" -v > "$evidence/02-log.txt" 2>&1
result=$?
printf "%s\n" "$result" > "$evidence/02-exit.txt"
exit "$result"
