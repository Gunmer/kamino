# Spec Delta

## MODIFIED Requirements

### Requirement: Actualización exclusiva del binario instalado

`kamino update` DEBE (`MUST`) actualizar únicamente un binario autocontenido en ejecución. Un binario Go de release DEBE (`MUST`) considerarse autocontenido. Una instalación desde código, `go run` o un binario Go sin la marca de autocontenido DEBE (`MUST`) fallar con un mensaje que indique la vía de actualización correspondiente, sin descargar ni escribir nada. Las instalaciones con `uv` o `pip` dejan de estar soportadas.

#### Scenario: Binario autocontenido

- **WHEN** se ejecuta `kamino update` desde el binario autocontenido, Go o Python legado
- **THEN** el comando actualiza el archivo ejecutable en ejecución

#### Scenario: Instalación desde código

- **WHEN** se ejecuta `kamino update` desde una instalación de código o un binario Go sin marca de autocontenido
- **THEN** el comando termina con error e indica la vía de actualización correspondiente
- **AND** no descarga ni escribe ningún archivo

#### Scenario: Construcción de desarrollo Go

- **GIVEN** un binario construido con `go run` o sin la marca de autocontenido
- **WHEN** se ejecuta `kamino update`
- **THEN** el comando termina con error y no descarga ni reemplaza el ejecutable

### Requirement: Errores de actualización

U1 DEBE (`MUST`) presentar el error común para una instalación no autocontenida, una plataforma no soportada, un fallo de red o un fallo de integridad. En la instalación no autocontenida DEBE indicar la vía de actualización correspondiente y NO DEBE descargar ni escribir.

#### Scenario: Instalación no autocontenida

- **GIVEN** una instalación desde código o un binario Go sin marca de autocontenido
- **WHEN** se ejecuta `kamino update`
- **THEN** se muestra la vía de actualización correspondiente y no se descarga ni escribe nada

#### Scenario: Plataforma no soportada

- **GIVEN** un sistema que no es macOS arm64
- **WHEN** se ejecuta `kamino update`
- **THEN** se muestra el error y no se descarga ni escribe nada

#### Scenario: Fallo de red o integridad

- **GIVEN** un fallo al consultar, descargar o verificar
- **WHEN** termina la operación
- **THEN** se muestra el error común, el binario actual sigue intacto y el temporal se elimina
