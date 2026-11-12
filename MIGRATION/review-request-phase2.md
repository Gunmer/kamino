# Solicitud de revisión — Fase 2

Revisar el contrato de pruebas antes de la implementación completa:

1. ¿Cada requisito de `MIGRATION/spec-coverage.md` tiene una prueba o una
   verificación razonable?
2. ¿Los fixtures dorados comparan el modelo semántico y no bytes de Python?
3. ¿La matriz cruzada cubre clon y holocrón en ambas direcciones?
4. ¿Alguna prueba toca el HOME real, `/Applications` o la red?
5. ¿El presupuesto de arranque mide el binario y no el proceso de test?

Puntos de atención conocidos:

- `golangci-lint`, `gofumpt`, `goreleaser` y `hyperfine` no están disponibles sin
  red; se usan `go vet`, `gofmt`, `go build` y arneses propios.
- La TUI se prueba a nivel de modelo; el arranque real se validó con un PTY.
- Los mensajes de parser pueden diferir entre Python y Go; el contrato es el
  estado (`valid`, `invalid`, `warning`, `unvalidated`).
