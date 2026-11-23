# Spec Delta

## ADDED Requirements

### Requirement: Instalación de apps ausentes en el despliegue

El despliegue DEBE (`MUST`) aceptar `--install-missing` para instalar las apps ausentes que tengan una receta aplicable. En modo no interactivo DEBE (`MUST`) exigirse junto con `--yes`; sin `--install-missing` el despliegue NO DEBE (`MUST NOT`) instalar nada y mantiene su comportamiento actual. Con `--dry-run` DEBE (`MUST`) mostrar qué se instalaría sin ejecutar nada.

#### Scenario: Sin el flag

- **GIVEN** un clon con una app ausente que tiene receta
- **WHEN** se ejecuta `kamino implant --yes` sin `--install-missing`
- **THEN** la app se informa como «Sin base en este Mac» y no se instala.

#### Scenario: Con el flag y confirmación

- **GIVEN** un clon con una app ausente que tiene receta aplicable
- **WHEN** se ejecuta `kamino implant --yes --install-missing`
- **THEN** la app se instala y su configuración se despliega.

#### Scenario: Flag sin confirmación

- **GIVEN** una ejecución no interactiva sin `--yes` y sin `--dry-run`
- **WHEN** se pasa `--install-missing`
- **THEN** el comando falla con código 1 pidiendo `--yes` y no instala nada.

#### Scenario: Ayuda

- **WHEN** se consulta la ayuda de `implant`
- **THEN** `--install-missing` aparece descrito en español.
