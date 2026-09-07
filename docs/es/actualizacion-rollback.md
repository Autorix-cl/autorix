# Actualización y rollback

Cada actualización debe tratarse como un cambio controlado. Antes de comenzar,
registre el commit, valores Helm, digests, versión de migración, aprobador y
responsable de rollback.

## Actualizar

1. Revise notas, valores y diff renderizado con el override de digests.
2. Respalde cada base de datos y verifique una restauración aislada.
3. Aplique migraciones mediante el hook del chart y despliegue el release.
4. Verifique health, OAuth, reglas de Aegis, administración de Console, métricas
   y entrega de alertas.

## Revertir

1. Detenga o drene el rollout si es seguro.
2. Vuelva al digest y valores previamente aprobados; nunca a un tag mutable.
3. No revierta el esquema de base de datos a ciegas. Restaure un backup probado
   solo si el plan lo permite y se aprueba el riesgo de pérdida de datos.
4. Repita smoke tests y registre incidente, evidencia y acciones posteriores.

Un rollback de Helm cambia manifiestos Kubernetes; no deshace mutaciones de
datos, rotación de claves ni efectos de terceros. Mantenga cambios de esquema
compatibles hasta cerrar la ventana de rollback.
