# Solicitud de revisión — Fase 3a

Revisar las invariantes de seguridad de `paths` y `archive`:

1. ¿Toda ruta intermedia se abre con `O_NOFOLLOW` y descriptor relativo?
2. ¿La extracción rechaza rutas absolutas, `..`, tipos especiales, enlaces
   absolutos o que salen del destino y miembros dentro de un enlace?
3. ¿El manifiesto mantiene el orden de campos, los `null` y la validación de
   `sha256`, `size`, `mode` y `link`?
4. ¿El commit atómico usa `link` sin sobrescritura y `rename` con revalidación?
5. ¿La cancelación elimina el temporal y deja intacto el clon anterior?
