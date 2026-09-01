#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
meta="$repo_root/.gooo/rewrite.gooo"

test "$(grep -c '^operator ' "$meta")" -eq 3
test "$(grep -c '^scenario ' "$meta")" -eq 7
grep -q '^authority metacode$' "$meta"
grep -q '^search bound=1 ' "$meta"
grep -q '^cross_project_gate required=0$' "$meta"
grep -q '^source_policy .*repository_writes=zero$' "$meta"
grep -q '^unknown_fields stage,step,reason,unknown_class,next_operation,blocked_by$' "$meta"
grep -q 'README.md' "$repo_root/contracts/metrics-v1.json"
if grep -nE 'git (commit|merge|push|reset|checkout)|gh (pr merge|release delete)' "$repo_root/scripts" "$repo_root/cmd"; then
  echo 'automatic source mutation or repository integration is forbidden' >&2
  exit 1
fi
echo 'semantic_audit=CLOSED'
