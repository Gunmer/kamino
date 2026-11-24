# Design

## Context

`internal/install` ya tiene tres métodos cerrados (`brew`, `installer`, `binary`). `Select` recorre las
recetas en orden, filtra por `os`/`arch` y salta `brew` si falta en el `PATH`; ese orden ya es la
preferencia. `binary` exige `url` y `sha256` fijos y un único destino, por lo que no sirve para
«última versión». Las escrituras bajo el HOME pasan por `paths.SafeHome` (no se degrada). Ver
`proposal.md` para la motivación.

Datos del estudio de los releases `latest` (macOS):

- Formatos: `.zip`, `.tar.gz` y binario suelto (`herdr`). Algunos assets llevan la versión en el nombre
  (`gh`, `lazygit`, `glow`, `bat`, `micro`) y otros no.
- Checksums publicados: gh, lazygit, glow, claude-code, copilot-cli, openspec, pi, micro (un `.sha` por
  asset) y codex. No publican: yazi, bat, opencode, gemini-cli, herdr.
- Paquetes npm: `@anthropic-ai/claude-code`, `@openai/codex`, `@google/gemini-cli`, `@github/copilot`,
  `opencode-ai`, `@fission-ai/openspec`, `@mariozechner/pi-coding-agent`.

## Goals / Non-Goals

**Goals:**
- Dos métodos nuevos dentro del conjunto cerrado, sin tocar el comportamiento de `binary`.
- Mantener el invariante «verificar antes de escribir» siempre que el release publique sumas.
- Que el orden del TOML siga siendo la única fuente de preferencia.

**Non-Goals:**
- Instalar o actualizar Homebrew, Node o npm.
- Fijar o elegir versiones; actualizar apps ya instaladas.
- Resolver el `PATH` del usuario tras instalar en `~/.local/bin`.

## Decisions

1. **`github-release` es un método nuevo, no una extensión de `binary`.** `binary` es un contrato
   publicado (URL + sha256 fijos); cambiarlo sería un cambio incompatible. Alternativa descartada:
   hacer opcional `sha256` en `binary`, que debilita el requisito existente.

2. **Una receta por arquitectura, sin mapa de arquitecturas.** Cada receta declara `arch` (campo ya
   existente) y el nombre literal del asset de esa arquitectura (`aarch64` / `x86_64` / `x64` varían
   por proyecto). Evita un sistema de plantillas por arquitectura. Coste: dos recetas por app.

3. **Resolución de «latest» por redirección, sin API.** Se hace una petición a
   `https://github.com/<repo>/releases/latest` y se lee la redirección `…/tag/<tag>`; la versión es el
   tag sin prefijo `v`. Con eso se descarga `…/releases/download/<tag>/<asset>`. La API REST anónima
   tiene un límite de 60 peticiones/hora por IP y exige parsear JSON; la redirección no. Se sigue
   usando `curl` con los mismos flags (`https` forzado) mediante el `Runner` existente, así los tests
   inyectan un runner.

4. **Extracción en Go, sin `unzip`/`tar` externos.** `archive/zip` y `archive/tar` con `compress/gzip`,
   leyendo solo los miembros nombrados en la receta (con `{version}` sustituible), solo ficheros
   regulares, sin seguir enlaces ni rutas absolutas o con `..`. Cada fichero se escribe con
   `SafeHome.CopyFile(dest, reader, 0o755)`, que ya es atómico. Antes de escribir ningún destino se
   comprueba que todos los miembros existen, para no dejar instalaciones a medias (yazi: `yazi` y `ya`).

5. **Checksums opcionales por receta (`checksums = "<nombre>"`).** Con `checksums`: se descarga ese
   fichero del mismo release, se busca la línea del asset (`<hash>  <nombre>`) y se compara con el
   SHA-256 calculado en streaming; falta, ausencia de línea o discrepancia ⇒ error sin escribir. Sin
   `checksums`: solo HTTPS + procedencia de GitHub, y la confirmación/`--dry-run` lo declara. Es una
   rebaja consciente respecto a `binary` para yazi, bat, opencode, gemini-cli y herdr. Alternativa
   descartada: no ofrecer esas apps sin brew, por pedido expreso de ofrecer «todas las recetas posibles».
   El fichero de sumas procede del mismo origen que el binario, así que detecta corrupción pero no un
   release comprometido; el spec no lo presenta como más.

6. **`npm` ejecuta `npm install --global <package>`.** Sin `sudo`, sin `--prefix` ni configuración
   propia: se respeta el `prefix` del usuario. `Select` generaliza la comprobación actual de `brew`
   a «el método necesita una herramienta»: `brew` → `brew`, `npm` → `npm`. `package` se valida con un
   patrón estricto (con `@scope/` opcional), sin espacios ni metacaracteres.

7. **Preferencia = orden del TOML, convención `brew`, `npm`, `installer`, `github-release`.** No se
   añade un campo de prioridad. Un test sobre el catálogo integrado fija ese orden y que cada receta es
   válida, para que no derive.

8. **Destino de los binarios: `.local/bin/<comando>`.** Es una ruta bajo el HOME aceptada por
   `SafeHome`. La detección posterior a la instalación (`isAppInstalled`) usa el `PATH`; si
   `~/.local/bin` no está en él, la app se informa «instalada pero no detectada», que ya es un estado
   existente de fallo no fatal (salida 3).

## Risks / Trade-offs

- [Sin checksums en 5 apps] → la confirmación y `--dry-run` lo muestran; el usuario puede declinar. El
  catálogo del usuario puede sobrescribir la receta por id.
- [Un release sin el asset esperado (renombrado, otra arquitectura)] → error claro «asset no
  encontrado», app omitida, salida 3; el resto del despliegue continúa.
- [Tag no estándar, p. ej. `rust-v0.171.0` en codex] → `{version}` solo se usa en assets que llevan la
  versión; codex y los demás con assets sin versión no dependen del formato del tag.
- [`npm install --global` falla por permisos (prefix de sistema)] → se reporta el error de npm y la
  app queda omitida; no se intenta `sudo`.
- [Rate limiting o indisponibilidad de github.com] → fallo no fatal por app.
- [Los nombres de assets cambian entre releases] → el catálogo se rompe silenciosamente hasta la
  siguiente edición; mitigado por el error explícito y por la posibilidad de corregir el TOML.
- [Superficie de seguridad ampliada: más descargas y extracción] → HTTPS forzado, solo `github.com`,
  extracción acotada a miembros nombrados y escritura por `SafeHome`.

## Open Questions

- Si `~/.local/bin` debe añadirse al `PATH` o solo advertirse al usuario: no cambia specs ni tareas
  (queda fuera de alcance) y puede tratarse después.
