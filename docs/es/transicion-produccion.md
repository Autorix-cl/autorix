# Transición a producción

Kubernetes/Helm es un modelo operativo distinto de Docker Compose. Probalo
primero en un entorno similar a producción.

1. Construí, escaneá, firmá y registrá digests inmutables. Helm de producción
   falla si falta un digest SHA-256 real.
2. Provisioná PostgreSQL con TLS, secretos por servicio, backups, monitoreo y
   restauración probada.
3. Entregá un archivo de valores de release con digests verificados, dominio,
   certificados, selectores de red y límites. No pongas contraseñas ahí.
4. Desplegá con Helm y verificá migraciones, health, administración privada,
   dashboards y alertas.
5. Ejecutá smoke test, ejercicio de restauración y rollback antes de aceptar
   tráfico.

Usá la [guía de Kubernetes](./production_k8s_guide.md) y el checklist de
aceptación como insumos. No constituyen certificación bancaria, aprobación
regulatoria ni evaluación externa de seguridad.
