# Fase 2 — Tests por capas y cobertura

## Contrato leído

Se revisaron `clone-safety`, `global-app-config`, `ghostty-config-references`,
`tui-navigation`, `clone-tui`, `deploy-conflicts`, `user-configuration`,
`cli-options`, `self-update`, `release-verification` y `tui-ansi-theme`, además de
`AGENTS.md`.

## Entregado

- Pruebas de `archive`, `paths`, `catalog`, `detect`, `plan`, `ghostty`,
  `snapshot`, `clone`, `conflicts`, `deploy`, `config`, `cli`, `update`, `ui` y
  `tui`.
- Compatibilidad cruzada Python/Go en `internal/integration`:
  - clon Python desplegado por Go;
  - clon Go desplegado por Python;
  - holocrón Go restaurado por Python.
- Fixtures doradas en `testdata/golden/` y presupuesto de arranque en
  `internal/performance`.
- `MIGRATION/spec-coverage.md` con una fila por cada uno de los 99 requisitos.

## Verificación

```sh
go vet ./...
go test ./... -race
go test ./internal/integration -run Cross -v
scripts/guard.sh
```

Las pruebas usan HOMEs temporales y catálogos inyectados; no leen el HOME real ni
`/Applications`. La matriz cruzada requiere `KAMINO_PYTHON` o un
`legacy/python/.venv` funcional.
