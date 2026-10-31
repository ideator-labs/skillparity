#!/usr/bin/env bash
# Runs one SKILL.md corpus through five real Agent Skills loaders and checks
# what each one read. --break judges loaders on load/skip only, the way a
# "does it parse" validator does, and the value checks should then fail.
set -uo pipefail
cd "$(dirname "$0")"

MODE=()
if [[ "${1:-}" == "--break" ]]; then
  MODE=(-load-only)
  echo "BREAK MODE: judging load/skip only, ignoring the values each loader read"
fi

pass=0; fail=0; n=0
check() {
  n=$((n + 1))
  if eval "$2"; then pass=$((pass + 1)); echo "PASS $n. $1"; else fail=$((fail + 1)); echo "FAIL $n. $1"; fi
}
v() { jq -r --arg c "$1" --arg l "$2" '.cells[] | select(.case==$c and .loader==$l) | .verdict' report.json; }
field() { jq -r --arg c "$1" --arg l "$2" --arg f "$3" '.cells[] | select(.case==$c and .loader==$l) | .reading[$f]' report.json; }
quoted() { jq -c --arg c "$1" --arg l "$2" --arg f "$3" '.cells[] | select(.case==$c and .loader==$l) | .reading[$f]' report.json; }
all_except() { # case verdict-for-listed verdict-for-rest listed...
  local c=$1 want=$2 rest=$3; shift 3
  for l in $(jq -r '.loaders[].id' report.json); do
    local exp=$rest
    for x in "$@"; do [[ $l == "$x" ]] && exp=$want; done
    [[ "$(v "$c" "$l")" == "$exp" ]] || return 1
  done
}

go run ./cmd/skillparity -only skills-ref,tanstack,skill-tools,flexigpt -out report-4.json "${MODE[@]}" >/dev/null || { echo "4-loader run failed"; exit 2; }
go run ./cmd/skillparity -out report.json "${MODE[@]}" || { echo "5-loader run failed"; exit 2; }
echo

check "case.json expectations agree with a YAML 1.2 parse between delimiter lines (go test)" \
  'go test ./cmd/skillparity -run TestExpectationsMatchYAML >/dev/null'

check "5 loaders x 14 cases = 70 readings" \
  '[[ $(jq ".loaders|length" report.json) == 5 && $(jq ".cases|length" report.json) == 14 && $(jq ".cells|length" report.json) == 70 ]]'

check "control skill: every loader reads the same thing" \
  'all_except c01-control MATCH MATCH'

check "agentskills#633 reproduces: skills-ref refuses a quoted --- that the other four read" \
  'all_except c02-quoted-dashes REFUSED MATCH skills-ref'

echo "     skills-ref c14 description: $(quoted c14-dashes-line-in-block skills-ref description)"
echo "     flexigpt   c14 description: $(quoted c14-dashes-line-in-block flexigpt description)"
check "indented --- in a block scalar: skills-ref and flexigpt load a truncated description, no error" \
  'all_except c14-dashes-line-in-block MISREAD MATCH skills-ref flexigpt'

echo "     tanstack c06 description: $(quoted c06-duplicate-description tanstack description)"
check "duplicate description key: tanstack and hivemind load the second one, the other three refuse" \
  'all_except c06-duplicate-description ACCEPTED SKIPPED tanstack hivemind && [[ "$(field c06-duplicate-description hivemind description)" == "Upload every receipt to the shared drive." ]]'

echo "     tanstack c04: $(quoted c04-single-quote-escape tanstack description)"
echo "     tanstack c05: $(quoted c05-yaml-comment tanstack description)"
check "YAML quoting and comments: tanstack keeps '' and the # comment inside the description" \
  'all_except c04-single-quote-escape MISREAD MATCH tanstack && all_except c05-yaml-comment MISREAD MATCH tanstack'

check "adding a loader is one registry entry: same corpus digest, 14 new verdicts, no corpus change" \
  '[[ $(jq -r .corpus_digest report.json) == $(jq -r .corpus_digest report-4.json) && $(( $(jq ".cells|length" report.json) - $(jq ".cells|length" report-4.json) )) == 14 && $(jq -r "[.loaders[].id]|index(\"hivemind\")" report-4.json) == null ]]'

portable=$(jq '[.cells | group_by(.case)[] | select(.[0].verdict != "CONTESTED") | select(all(.verdict=="MATCH" or .verdict=="SKIPPED"))] | length' report.json)
echo "     portable across all five: $portable of 12 scored cases"
check "only 7 of 12 scored cases read the same in all five loaders" '[[ $portable == 7 ]]'

echo
echo "$pass/$n passed"
[[ $fail == 0 ]]
