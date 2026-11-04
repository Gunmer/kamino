# Diseño: alias cortos para las opciones de la CLI

## Contexto

`src/kamino/cli.py` declara opciones de Typer para `backup`, `restore` y el callback principal de versión. La ayuda y las opciones de autocompletado las genera el framework. El motivo y el mapa de alias están en `proposal.md`; los comportamientos exigidos están en `specs/cli-options/spec.md`.

La restauración selecciona aplicaciones cuando hay terminal y no se usa `--yes`, incluso en una simulación. Sin terminal, rechaza una restauración sin confirmar, excepto si es una simulación. Las pruebas de los casos de uso ya verifican que `dry_run` no escribe archivos ni instantáneas en directorios temporales.

La verificación integrada detectó un fallo previo en `tests/test_release.py`: el repositorio simulado comienza en `v0.1.0`, pero copia el `CHANGELOG.md` real, que ya contiene `v0.2.0`. La plantilla evita duplicar esa sección y no genera la entrada que la prueba espera. El fixture también copia una versión del código real que puede no coincidir con la versión inicial simulada.

## Objetivos y exclusiones

**Objetivos:** incorporar los alias en la capa de CLI, mantener una única fuente de comportamiento por opción, verificar la ayuda en los tres niveles y corregir el aislamiento de la prueba de publicación que bloquea la verificación completa.

**Exclusiones:** modificar los casos de uso, la selección interactiva, el formato del respaldo, la configuración de autocompletado, las políticas de seguridad y códigos de salida existentes o la configuración y el historial de publicación reales.

## Decisiones

### Usar los alias nativos de Typer

Añadir cada nombre corto a la misma declaración `typer.Option` que su nombre largo, por ejemplo `typer.Option("--apps", "-a", ...)`. Declarar los alias compartidos en ambas opciones existentes de `backup` y `restore`; reservar `-V` para la versión principal y `-v` para el detalle de cada comando.

Se descarta preprocesar argumentos o crear comandos nuevos: ambas alternativas duplicarían la interpretación y podrían divergir de las opciones actuales.

### Configurar la ayuda en el contexto principal

Configurar `help_option_names` con `-h` y `--help` en el contexto de la aplicación, y comprobar la ayuda principal y la de ambos comandos. Una comprobación en memoria con la versión instalada de Typer confirmó que los subcomandos heredan esta configuración; las pruebas deben detectar cualquier diferencia de comportamiento en otras versiones compatibles.

Se descarta configurar cada subcomando por separado mientras la herencia satisfaga las pruebas, para evitar definiciones duplicadas. Los nombres de las opciones automáticas de autocompletado no se personalizan.

### Conservar los límites de seguridad

Los alias deben alimentar los parámetros ya existentes: `-n` transmite `dry_run=True` y `-y` transmite la misma confirmación que `--yes`. No se modifica el orden de validación, verificación, instantánea y sustitución de la restauración. En particular, `-n` no debe suprimir la selección interactiva que actualmente se realiza sin `--yes`.

### Probar en la frontera de la CLI

Ampliar `tests/test_cli.py` con `CliRunner` y dobles de detección, catálogo, casos de uso e interacción. Comparar los argumentos y resultados de las formas cortas y largas, incluida la confirmación sin terminal, la simulación, el detalle y los códigos 1 y 3. Verificar que ayuda y versión terminan sin despachar el menú ni los casos de uso, y que la ayuda muestra ambos nombres sin depender del formato exacto de Rich.

Mantener las pruebas existentes de `tests/test_backup.py` y `tests/test_restore.py` como verificación de que la simulación no escribe: usan HOME y directorios de instantáneas temporales. No invocar la instalación real de autocompletado ni leer el HOME real o `/Applications` en las pruebas de CLI.

### Aislar el estado inicial de las pruebas de publicación

Inicializar el repositorio temporal de `tests/test_release.py` con un historial sintético propio que termine en `v0.1.0` y con la misma versión en la etiqueta, `pyproject.toml`, `src/kamino/__init__.py` y `uv.lock`. La siguiente versión simulada, `v0.2.0`, no debe existir en ese historial. El fixture no debe copiar el historial ni la versión del código reales, que cambian entre releases; sí debe conservar las plantillas y la configuración de publicación que se quieren verificar.

Comprobar la coherencia inicial y la independencia frente a versiones e historiales reales distintos, sin modificar los archivos reales. Conservar las comprobaciones de la versión generada, la nota de cambio incompatible, el historial previo, las notas de la publicación, el archivo de dependencias y la repetición sin cambios; comprobar además que la nota pertenece a la sección nueva. Las pruebas siguen usando HOME y repositorios temporales, sin publicar ni contactar remotos.

## Riesgos y compensaciones

- **Confusión entre `-v` y `-V`** → describir ambos en la ayuda y el README y probar su ámbito y comportamiento.
- **Herencia distinta de la ayuda en alguna versión compatible de Typer** → probar `-h` y `--help` en los tres niveles; si la herencia falla, configurar explícitamente los comandos afectados sin alterar la interfaz.
- **Una simulación o confirmación podría cambiar de semántica** → comparar parámetros y ramas interactivas/no interactivas en la CLI y conservar las pruebas de no escritura y seguridad de los casos de uso.
- **Los datos reales pueden volver a contaminar el repositorio de prueba** → usar datos sintéticos y añadir una regresión que detecte versiones e historiales incompatibles con la base simulada.

## Despliegue y reversión

El cambio es aditivo y no requiere migración de datos. Se distribuye mediante el proceso normal del proyecto. Una reversión puede retirar juntos los alias, sus pruebas y la documentación; aunque las formas largas permanecerían disponibles, los scripts que ya hayan adoptado las formas cortas dejarían de funcionar y tendrían que volver a las largas.
