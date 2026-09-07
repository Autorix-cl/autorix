# Transición a producción

Kubernetes/Helm es un modelo operativo distinto de Docker Compose. Debe
validarse primero en un entorno similar a producción.

1. Construya, analice, firme y registre digests inmutables. Helm de producción
   falla si falta un digest SHA-256 real.
2. Aprovisione PostgreSQL con TLS, secretos por servicio, backups, monitoreo y
   restauración probada.
3. Entregue un archivo de valores de release con digests verificados, dominio,
   certificados, selectores de red y límites. No incluya contraseñas.
4. Despliegue con Helm y verifique migraciones, health, administración privada,
   dashboards y alertas.
5. Ejecute smoke tests, un ejercicio de restauración y rollback antes de aceptar
   tráfico.

Utilice la [guía de Kubernetes](./production_k8s_guide.md) y el checklist de
aceptación como insumos. No constituyen certificación bancaria, aprobación
regulatoria ni evaluación externa de seguridad.
