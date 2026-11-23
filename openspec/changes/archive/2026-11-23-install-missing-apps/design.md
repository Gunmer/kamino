# Design

## Context

Al preparar un despliegue, `deploy.PrepareDeploy` separa las apps del clon en `DeployableApps` y `NotInstalledApps` según `isAppInstalled` y los detectores del catálogo. Las ausentes solo se informan. El catálogo (`internal/catalog`, `AppDefinition` con `Validate`) carga TOML embebidos y de usuario; el de usuario reemplaza al integrado con el mismo id. `internal/plan` recorre las rutas declaradas para clonar. `internal/update` ya descarga y verifica SHA-256 con un patrón propio.

Restricciones: las pruebas inyectan `Home`, `Catalog`, `ApplicationDirs` y `SnapshotsDir` y nunca tocan el HOME real ni `/Applications`; el orden del despliegue (validar manifiesto, verificar archivo, instantánea, reemplazos atómicos) no cambia; `--dry-run` no escribe nada.

## Goals / Non-Goals

**Goals:**
- Esquema cerrado de tres métodos, validado al cargar el catálogo.
- Ningún dato del clon influye en qué se descarga o ejecuta.
- Ejecución de procesos y descargas inyectables, para probar sin red ni instalar nada.

**Non-Goals:**
- Recetas en las apps integradas (primera versión: solo el mecanismo).
- Instalar Homebrew, desinstalar apps o actualizarlas.
- Verificar firmas o notarización de binarios más allá del SHA-256.
- Soporte de plataformas distintas de las que declare cada receta.

## Decisions

**1. Paquete nuevo `internal/install`.** Contiene el tipo `Recipe`, la selección de receta por plataforma y la ejecución de cada método. `internal/catalog` solo parsea y valida el bloque `[[install]]`; `internal/deploy` orquesta. Alternativa descartada: meter la ejecución en `internal/deploy`, que ya concentra clasificación y aplicación y mezclaría procesos externos con reemplazos atómicos.

**2. Receta solo en `AppDefinition`.** El manifiesto no se toca y no sube de versión. Así un clon ajeno no puede dictar URLs. Alternativa descartada: recetas en el clon (portable, pero equivale a ejecutar remoto por encargo de terceros).

**3. Lista ordenada con selección en tiempo de uso.** Se filtra por `os`/`arch` y, para `brew`, por la presencia de `brew` en el `PATH` inyectado. Se usa la primera receta viable; sin ninguna, la app no se ofrece (por plataforma) o se omite (por herramienta ausente en la instalación).

**4. `binary` verificado y atómico.** Descarga con `curl --fail --proto =https --location` a un temporal dentro del HOME, calcula SHA-256 en streaming (como `internal/update`), y solo si coincide lo mueve al destino con `rename` y permisos 0755. El destino se resuelve con `internal/paths.SafeHome` (`openat`/`O_NOFOLLOW`), sin degradar a comprobaciones léxicas. El temporal se borra siempre.

**5. `installer` sin tubería.** Se descarga a un temporal, se muestra la URL en la confirmación y se ejecuta `sh <temporal>`. Sin SHA-256, porque los instaladores oficiales cambian; la mitigación es HTTPS, confirmación con URL y origen en el catálogo local. Alternativa descartada: `curl | sh`, porque impide inspeccionar o abortar.

**6. Fase de instalación en `DeploySession`.** Nueva etapa entre `PrepareDeploy` y `Apply`: tras verificar el archivo completo y antes de la instantánea. Cada instalación correcta mueve la app de `NotInstalledApps` a `DeployableApps` y se vuelve a clasificar con los detectores. Una instalación fallida la deja en omitidas (código 3) y no aborta.

**7. Activación.** La TUI añade una pantalla de selección entre la de apps y la de confirmación, solo si hay apps instalables. La CLI añade `--install-missing`, que exige `--yes` en modo no interactivo. `--dry-run` calcula y muestra el plan de instalación sin llamar al ejecutor.

**8. Guardia contra binarios en `internal/plan`.** Al recorrer rutas, se rechazan directorios `.app` y archivos con bit de ejecución o formato ejecutable (Mach-O), se informan como no clonados y no se leen. Es independiente del catálogo para cubrir definiciones de usuario.

**9. Ejecutor inyectable.** `install.Runner` envuelve procesos (`brew`, `curl`, `sh`) y búsqueda en el `PATH`. Las pruebas usan un fake y un servidor HTTPS local; ninguna prueba ejecuta instaladores reales.

## Risks / Trade-offs

- [Ejecución de código remoto con `installer`] → solo HTTPS, receta desde el catálogo local, confirmación con URL exacta, flag explícito en CLI, ejecución desde temporal.
- [Catálogo de usuario malicioso o editado] → el catálogo ya es de confianza del usuario; `apps validate` informa de las recetas y la confirmación muestra el método y la URL.
- [SHA-256 caduca cuando el proveedor actualiza el binario] → fallo seguro: la instalación se omite con un mensaje claro; las recetas se mantienen en el catálogo.
- [Una instalación parcial deja residuos] → temporal siempre borrado; `rename` atómico del binario; `brew` e `installer` gestionan su propio estado.
- [Falsos positivos de la guardia de binarios] → se informa cada rechazo con su ruta para que el usuario lo vea.
- [Instalar antes de la instantánea] → una app recién instalada puede crear su configuración por defecto; la instantánea la captura después, como cualquier destino que cambia.

## Migration Plan

Cambio aditivo: sin recetas no cambia ningún comportamiento. Sin migración de datos ni del manifiesto. Para revertir basta retirar el flag y la etapa; los clones y catálogos existentes siguen siendo válidos (las claves `[[install]]` desconocidas fallarían en versiones antiguas, por lo que no se añaden recetas integradas en esta versión).

## Open Questions

- Ubicación exacta del destino de `binary` (`dest` obligatorio o por defecto `.local/bin/<command>`); se decide al implementar sin cambiar los requisitos.
