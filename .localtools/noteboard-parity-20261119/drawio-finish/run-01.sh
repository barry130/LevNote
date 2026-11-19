#!/usr/bin/env bash
set -uo pipefail
cd /f/AIproject/LevNote || exit 2
evidence=.localtools/noteboard-parity-20261119/drawio-finish
command_text="bash .localtools/gate.sh test -count=1 -timeout=300s ./modules/graphview -run DrawioClone -v"
printf "%s\n" "$command_text" > "$evidence/01-command.txt"
bash .localtools/gate.sh test -count=1 -timeout=300s ./modules/graphview -run DrawioClone -v > "$evidence/01-log.txt" 2>&1
result=$?
printf "%s\n" "$result" > "$evidence/01-exit.txt"
exit "$result"
