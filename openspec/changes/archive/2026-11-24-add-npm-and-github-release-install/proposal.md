# Proposal

## Why

Hoy casi todas las apps del catálogo solo se instalan con Homebrew; si el usuario no lo tiene, la app
no se ofrece aunque tenga `npm` o pueda descargarse el binario desde GitHub. El método `binary`
exige URL y `sha256` fijos, así que no sirve para «siempre la última versión» de un release.

## What Changes

- Nuevo método de receta `npm`: instala un paquete global con `npm install --global`, siempre en su
  última versión. Se omite si `npm` no está en el `PATH`.
- Nuevo método de receta `github-release`: descarga el binario del último release de un repositorio de
  GitHub, lo extrae (zip, tar.gz o binario suelto) y lo instala bajo el HOME con permisos 0755.
- Verificación opcional por receta con el fichero de checksums del mismo release. Si la receta lo
  declara y no coincide (o no existe), no se instala nada. Sin él, solo HTTPS y procedencia de GitHub;
  la confirmación y `--dry-run` lo indican.
- Preferencia por método = orden de las recetas en el catálogo: `brew`, `npm`, `installer`,
  `github-release`. Un método cuya herramienta falta se salta y, si ninguno puede ejecutarse, la app
  no se ofrece.
- Se añaden al catálogo integrado todas las recetas posibles (brew, npm, instalador y release de
  GitHub) para cada app que las admita.
- **BREAKING**: ninguno. `binary`, `brew` e `installer` no cambian.

## Capabilities

### New Capabilities

### Modified Capabilities
- `app-install`: se amplía el conjunto cerrado de métodos con `npm` y `github-release`, y se añaden la
  verificación de checksums, la preferencia entre métodos y la cobertura del catálogo integrado.

## Impact

- Código: `internal/install` (`recipe.go`, `installer.go`, `runner.go`), validación en
  `internal/catalog` y confirmación en `internal/implant` / `internal/tui`.
- Catálogo: `internal/catalog/apps/*.toml` (15 apps ganan recetas).
- Docs: `docs/guia/catalogo.md`, `docs/tecnico/` y la ayuda de `apps validate`.
- Red: nuevas descargas desde `github.com` y `registry.npmjs.org` (esta última vía `npm`).
- Fuera de alcance: instalar `npm` o Node, fijar versiones, apps sin binario en releases (`git`,
  `oh-my-zsh`, `ghostty`) y Linux/Windows.
