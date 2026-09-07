# Comunidad, soporte y política de versiones

Autorix es un proyecto IAM autoalojado bajo licencia Apache-2.0. Se mantiene
con soporte comunitario de mejor esfuerzo; no es un servicio gestionado ni
ofrece un SLA.

## Obtener ayuda

| Necesidad | Canal |
| --- | --- |
| Instalación, configuración y uso | [GitHub Discussions](https://github.com/Autorix-cl/autorix/discussions) |
| Defecto reproducible | [GitHub Issues](https://github.com/Autorix-cl/autorix/issues/new/choose) |
| Vulnerabilidad de seguridad | [Aviso de seguridad privado](https://github.com/Autorix-cl/autorix/security/advisories/new) |

No publique secretos, datos personales ni detalles de vulnerabilidades en
canales públicos. Consulte [SECURITY.md](https://github.com/Autorix-cl/autorix/blob/main/SECURITY.md)
antes de informar un problema de seguridad.

## Versiones y releases

Autorix utiliza [versionado semántico](https://semver.org/spec/v2.0.0.html):

- Las versiones **mayores** pueden incluir cambios incompatibles y deben
  proporcionar notas de migración.
- Las versiones **menores** añaden capacidades compatibles hacia atrás.
- Las versiones de **parche** incluyen correcciones compatibles hacia atrás.

Un tag Git firmado y su GitHub Release son el registro autoritativo de una
versión. Use digests inmutables de imágenes en producción, revise las notas del
release antes de actualizar y siga la [guía de actualización y rollback](./actualizacion-rollback.md).
El [CHANGELOG](https://github.com/Autorix-cl/autorix/blob/main/CHANGELOG.md)
registra los cambios relevantes.

## Limitaciones conocidas

- El enrutamiento de alertas y la entrega de notificaciones externas requieren
  una configuración administrada por el operador para Prometheus y
  Alertmanager. Autorix no proporciona monitoreo gestionado ni cobertura de
  guardia.
- Docker Compose se usa para desarrollo local y evaluación; no debe exponerse a
  Internet ni utilizarse como despliegue de producción. Para planificar
  producción, use la guía de Kubernetes y complete el
  [checklist de aceptación para producción](./checklist-aceptacion-produccion.md).
- Autorix no está certificado para ISO 27001, SOC 2, PCI DSS ni estándares
  bancarios. Cada organización operadora debe realizar sus propias aprobaciones
  de seguridad, cumplimiento y operación.

## Contribuir

Revise [CONTRIBUTING.md](https://github.com/Autorix-cl/autorix/blob/main/CONTRIBUTING.md)
y el [Código de conducta](https://github.com/Autorix-cl/autorix/blob/main/CODE_OF_CONDUCT.md)
antes de abrir una contribución. La documentación pública del producto se
mantiene en inglés y español técnico neutro.
