---
description: Archiva un cambio OpenSpec completado
mode: primary
model: openrouter/anthropic/claude-haiku-5.5#high
steps: 15
permissions:
  - action: edit
    resource: "*"
    effect: deny
  - action: edit
    resource: "openspec/**"
    effect: allow
  - action: shell
    resource: "*"
    effect: ask
---
