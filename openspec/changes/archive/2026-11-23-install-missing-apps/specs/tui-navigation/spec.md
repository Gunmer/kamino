# Spec Delta

## ADDED Requirements

### Requirement: Selección de apps ausentes a instalar

Al implantar, si hay apps ausentes con receta aplicable, la TUI DEBE (`MUST`) preguntar por cada una si se instala antes de aplicar. La pregunta DEBE (`MUST`) mostrar el método y, para `installer`, la URL exacta. Rechazar una app NO DEBE (`MUST NOT`) cancelar el despliegue y la deja como «Sin base en este Mac».

#### Scenario: Instalar una app ausente

- **GIVEN** un clon con una app ausente con receta `brew`
- **WHEN** el usuario acepta instalarla
- **THEN** se instala antes de aplicar y su configuración se despliega.

#### Scenario: Rechazar la instalación

- **GIVEN** una app ausente con receta aplicable
- **WHEN** el usuario rechaza instalarla
- **THEN** la app se informa como «Sin base en este Mac» y el resto del despliegue continúa.

#### Scenario: Sin apps instalables

- **GIVEN** un clon cuyas apps ausentes no tienen receta aplicable
- **WHEN** se abre el flujo de implantación
- **THEN** no aparece la pregunta de instalación.
