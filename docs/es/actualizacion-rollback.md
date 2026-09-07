# Actualización y rollback

Tratà cada actualización como un cambio controlado. Antes de empezar, registrá
commit, valores Helm, digests, versión de migración, aprobador y responsable de
rollback.

## Actualizar

1. Revisá notas, valores y diff renderizado con el override de digests.
2. Respaldá cada base de datos y verificá una restauración aislada.
3. Aplicá migraciones mediante el hook del chart y desplegá el release.
4. Verificá health, OAuth, reglas de Aegis, administración de Console, métricas
   y entrega de alertas.

## Revertir

1. Detené o drená el rollout si es seguro.
2. Volvé al digest y valores previamente aprobados; nunca a un tag mutable.
3. No reviertas esquema de base de datos a ciegas. Restaurá un backup probado
   solo si el plan lo permite y se aprueba el riesgo de pérdida de datos.
4. Repetí smoke tests y registrá incidente, evidencia y acciones posteriores.

Un rollback de Helm cambia manifiestos Kubernetes; no deshace mutaciones de
datos, rotación de claves ni efectos de terceros. Mantené cambios de esquema
compatibles hasta cerrar la ventana de rollback.
