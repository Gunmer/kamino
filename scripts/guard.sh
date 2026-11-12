#!/bin/sh
# Falla si cambian los fixtures dorados o los tests de la Fase 2 sin una
# decisión registrada en MIGRATION/DECISIONS.md.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
base=${1:-main}

cd "$root"
list_changes() {
    if git rev-parse --verify "$base" >/dev/null 2>&1; then
        git diff --name-only "$base" -- "$@" 2>/dev/null || true
    else
        git diff --name-only -- "$@" 2>/dev/null || true
    fi
    git ls-files --others --exclude-standard -- "$@" 2>/dev/null || true
}

changed=$(list_changes testdata/golden 'internal/**/*_test.go' 'tests/**/*_test.go' | sort -u)
decisions=$(list_changes MIGRATION/DECISIONS.md | sort -u)

if [ -z "$changed" ]; then
    echo "guard: sin cambios en fixtures ni tests protegidos"
    exit 0
fi

if [ -z "$decisions" ]; then
    echo "guard: los fixtures o tests de la Fase 2 cambiaron sin una entrada en MIGRATION/DECISIONS.md" >&2
    echo "$changed" >&2
    exit 1
fi

echo "guard: cambios protegidos documentados en MIGRATION/DECISIONS.md"
