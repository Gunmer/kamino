#!/bin/sh
# Falla si cambian los fixtures dorados o los tests de la Fase 2 sin una
# decisión nueva registrada en MIGRATION/DECISIONS.md.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
base=${1:-}

cd "$root"
if [ -z "$base" ]; then
    for candidate in migration/phase-2 migration/phase-1; do
        if git rev-parse --verify "$candidate" >/dev/null 2>&1; then
            base=$candidate
            break
        fi
    done
fi
if [ -z "$base" ]; then
    base=HEAD
fi

protected_paths="testdata/golden internal/**/*_test.go tests/**/*_test.go"
committed=$(git diff --name-only "$base" -- $protected_paths 2>/dev/null || true)
working=$(git status --porcelain -- $protected_paths 2>/dev/null | awk '{print $2}' || true)
changed=$(printf '%s\n%s\n' "$committed" "$working" | sed '/^$/d' | sort -u)

if [ -z "$changed" ]; then
    echo "guard: sin cambios en fixtures ni tests protegidos"
    exit 0
fi

decisions_committed=$(git diff "$base" -- MIGRATION/DECISIONS.md 2>/dev/null | grep -c '^+## D' || true)
decisions_working=$(git diff -- MIGRATION/DECISIONS.md 2>/dev/null | grep -c '^+## D' || true)
decisions_staged=$(git diff --cached -- MIGRATION/DECISIONS.md 2>/dev/null | grep -c '^+## D' || true)
decisions_new=0
for count in "$decisions_committed" "$decisions_working" "$decisions_staged"; do
    if [ "${count:-0}" -gt 0 ]; then
        decisions_new=1
    fi
done

if [ "$decisions_new" -eq 0 ]; then
    echo "guard: los fixtures o tests de la Fase 2 cambiaron sin una entrada nueva en MIGRATION/DECISIONS.md" >&2
    echo "$changed" >&2
    exit 1
fi

echo "guard: cambios protegidos documentados en MIGRATION/DECISIONS.md"
