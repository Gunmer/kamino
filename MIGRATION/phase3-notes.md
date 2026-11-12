# Fase 3a — paths, archive y progress

## Implementado

- `internal/progress`: fases, eventos, callbacks y cancelación cooperativa con
  `context.Context`.
- `internal/paths`: `SafeHome` con `openat`, `O_NOFOLLOW`, `fstatat`,
  `readlinkat`, `mkdirat`, `renameat`, `symlinkat` y escrituras atómicas.
- `internal/archive`: PAX, gzip determinista, manifiesto JSON compatible,
  checksums, lectura acotada, verificación completa y extracción segura.

## Verificación

```sh
go test ./internal/paths ./internal/archive ./internal/progress
go vet ./...
```

`scripts/guard.sh` falla si cambian los fixtures o los tests protegidos sin una
decisión nueva en `MIGRATION/DECISIONS.md`.
