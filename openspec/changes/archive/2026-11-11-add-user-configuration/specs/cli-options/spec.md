# Spec Delta

## MODIFIED Requirements

### Requirement: Nombre por defecto y reemplazo del clon

En la vía no interactiva, `clone` DEBE (`MUST`) usar `clone-yyyyMMdd.tar.gz` con la fecha local en `~/.config/kamino/vault/`, creando ese directorio si no existe. Si el clon existe y no se indicó `--force`, DEBE terminar con código 1 y un error literal que indique cómo reemplazarlo. Con `--force`, DEBE reemplazarlo de forma atómica. Con `--dry-run`, NO DEBE crear ni reemplazar nada, incluido el directorio por defecto.

#### Scenario: Nombre por defecto sin TUI

- **GIVEN** un HOME sin `~/.config/kamino/vault/` ni un clon para la fecha local
- **WHEN** se ejecuta `kamino clone` sin `--output`
- **THEN** se crea `~/.config/kamino/vault/` y el archivo `~/.config/kamino/vault/clone-yyyyMMdd.tar.gz` con la fecha local

#### Scenario: Clon existente sin fuerza

- **GIVEN** un clon existente en el nombre por defecto
- **WHEN** se ejecuta `kamino clone` sin `--force`
- **THEN** el comando termina con código 1, no sobrescribe y explica que use `--force`

#### Scenario: Reemplazo con fuerza

- **GIVEN** un clon existente y un clon nuevo válido
- **WHEN** se ejecuta `kamino clone --force`
- **THEN** el clon anterior se reemplaza de forma atómica y no queda un temporal

#### Scenario: Simulación no escribe

- **GIVEN** un clon existente en el destino por defecto y `--force`
- **WHEN** se ejecuta `kamino clone --dry-run --force`
- **THEN** no se crea, reemplaza ni modifica ningún archivo

#### Scenario: Simulación no crea el directorio por defecto

- **GIVEN** un HOME sin `~/.config/kamino/vault/`
- **WHEN** se ejecuta `kamino clone --dry-run`
- **THEN** el directorio no se crea y el comando no escribe nada
