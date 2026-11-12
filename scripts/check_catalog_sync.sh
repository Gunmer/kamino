#!/bin/sh
# Falla si la copia embebida del catálogo difiere de src/kamino/catalog/apps.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
source_dir="$root/legacy/python/src/kamino/catalog/apps"
embedded_dir="$root/internal/catalog/apps"
if [ ! -d "$source_dir" ]; then
    echo "no existe $source_dir" >&2
    exit 1
fi
if ! diff -r "$source_dir" "$embedded_dir" >/dev/null 2>&1; then
    echo "el catálogo embebido no coincide con $source_dir" >&2
    diff -r "$source_dir" "$embedded_dir" >&2 || true
    exit 1
fi
echo "catálogo sincronizado"
