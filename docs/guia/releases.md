# Versionado y releases

El versionado sigue [SemVer](https://semver.org/) y se automatiza con `cmd/release`:

1. Los commits integrados en `main` deben seguir Conventional Commits. Con merge tradicional o
   rebase se analizan los commits de la rama; con squash, conserva un mensaje Conventional Commit
   adecuado en el commit resultante. CI no valida los títulos de los PRs.
2. Después de que CI pase en `main`, se calcula la siguiente versión desde el último tag:
   `feat:` aumenta minor, `fix:` aumenta patch y `!` o un footer `BREAKING CHANGE:` identifica
   cambios incompatibles. Si solo hay `docs:`, `ci:`, `chore:`, `perf:` u otros cambios sin ruptura
   de compatibilidad, no se genera versión ni release.
3. Durante las versiones `0.x`, los cambios incompatibles aumentan minor. Desde `1.0.0`,
   aumentan major. Se toma el mayor incremento entre los commits pendientes.
4. Se actualizan `CHANGELOG.md`, `internal/version/version.go` y el tag `vX.Y.Z`. El binario Go se
   construye y se comprueba antes de empujar el commit de release.
5. Se crea una GitHub Release con las notas y el binario autocontenido `kamino-arm64`
   (Go estático, Apple Silicon), incluido el catálogo de aplicaciones. No se distribuye actualmente
   un binario Intel. La release permanece en borrador hasta subir el binario.

Por ejemplo, desde `0.1.0`, `fix: corrige la implantación` produce `0.1.1`, mientras que
`feat: añade una aplicación` o `feat!: cambia el formato` produce `0.2.0`. Un merge que solo
contenga `ci: ajusta el workflow` no publica nada.

Las publicaciones se serializan. Si `main` avanza durante la construcción, se cancela el push
sin forzarlo; la ejecución del nuevo commit validado retomará los cambios pendientes. Los pushes
y tags del bot no disparan otro workflow: todo el proceso ocurre en **Release**.

Si falla la publicación después de empujar el tag, ejecuta manualmente **Release** desde la rama
`main` e indica ese tag en el campo `tag`. Esta recuperación reconstruye y comprueba la misma
versión y completa la release sin incrementar la versión, duplicar el changelog ni reemplazar
assets ya subidos. Si GitHub dejó un asset vacío por una subida fallida, lo elimina antes de
reintentar. No permite publicar tags ajenos a un commit de release integrado en `main`.

Los binarios descargados manualmente no están firmados y macOS puede aplicarles cuarentena. Para
quitarla manualmente:

```bash
xattr -d com.apple.quarantine ./kamino-arm64
```

El instalador intenta quitar ese atributo automáticamente.

El Homebrew tap se publicará tras la primera versión etiquetada.
