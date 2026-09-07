# Checklist de aceptación para producción

Un despliegue puede entrar a una ventana de cambio productiva solo si cada punto **obligatorio** aplicable tiene responsable, evidencia fechada y aceptación de riesgo registrada. Es una puerta de ingeniería, no certificación ni afirmación de idoneidad bancaria.

## Ruta rápida

1. Cree un registro de release con commit exacto, imágenes, valores Helm, entorno, aprobador y responsable de rollback.
2. Adjunte evidencia de cada control obligatorio; bloquee el release pendiente o documente aceptación de riesgo por la organización responsable.
3. Ejecute despliegue y rollback en un entorno similar a producción y registre sus resultados.

## Identidad del release y cadena de suministro

- [ ] **Obligatorio:** El registro contiene commit, fecha de build, digest de imagen, SBOM y análisis de vulnerabilidades.
- [ ] **Obligatorio:** Las imágenes están firmadas y la política de admisión valida firmante y digest antes del despliegue.
- [ ] **Obligatorio:** Hallazgos críticos/altos se corrigen o tienen excepción aprobada, acotada en tiempo y con controles compensatorios.
- [ ] **Obligatorio:** CI cubre pruebas unitarias, integración, race, configuración y render Helm pertinentes.
- [ ] **Obligatorio:** Un revisor independiente aprueba el diff productivo y el registro de cambio.

## Identidad, autorización y criptografía

- [ ] **Obligatorio:** Emisor, redirect URI, orígenes CORS, audiencias y upstreams Aegis están en allowlist y revisados.
- [ ] **Obligatorio:** Los listeners administrativos son privados y están protegidos por identidad, autorización, red y auditoría.
- [ ] **Obligatorio:** Claves, credenciales, certificados y material de cifrado provienen de un sistema de secretos aprobado; no hay secretos de desarrollo.
- [ ] **Obligatorio:** Se ensayaron rotación de claves, revocación de emergencia y expiración de caché JWKS sin perder continuidad de verificación.
- [ ] **Obligatorio:** El acceso privilegiado aplica mínimo privilegio, MFA, revisión periódica y procedimiento break-glass probado.

## Plataforma y protección de datos

- [ ] **Obligatorio:** La base de datos usa validación de certificado, cuentas de mínimo privilegio, cifrado, backups probados y retención definida.
- [ ] **Obligatorio:** Se validaron admisión Kubernetes, namespaces, service accounts, NetworkPolicies, pod security, cuotas y egress en el clúster destino.
- [ ] **Obligatorio:** TLS de ingress, HSTS, rate limit, WAF/DDoS, renovación de certificado y sus responsables están documentados y probados.
- [ ] **Obligatorio:** El dueño de datos aprobó límites de tenant, residencia, retención, eliminación y cifrado.
- [ ] **Obligatorio:** Ninguna carga depende de estado local mutable sin volumen persistente, backup, recuperación y responsable documentados.

## Resiliencia, monitoreo e incidentes

- [ ] **Obligatorio:** Objetivos de disponibilidad, RTO, RPO, alertas, on-call y escalamiento están aprobados.
- [ ] **Obligatorio:** Se restauró un backup cifrado en ambiente similar a producción y se cumplió RTO/RPO.
- [ ] **Obligatorio:** Dashboards y alertas cubren fallas de autenticación/autorización, ciclo de token/clave, base de datos, saturación e integridad de auditoría.
- [ ] **Obligatorio:** Los logs están centralizados, protegidos, sincronizados en tiempo, retenidos por política y probados para investigación.
- [ ] **Obligatorio:** Un tabletop cubre compromiso de credenciales, caída, exposición de datos y rollback; las acciones tienen seguimiento.

## Aseguramiento y control de cambio

- [ ] **Obligatorio:** Hay modelo de amenazas actual para endpoints públicos, administración privada, límites de confianza, flujos y abusos.
- [ ] **Obligatorio:** Un pentest independiente calificado cubre arquitectura y configuración destino; los hallazgos se corrigen o aceptan formalmente.
- [ ] **Obligatorio:** Las revisiones legal, privacidad, regulación y terceros exigidas por la organización están completas.
- [ ] **Obligatorio:** Existe plan de rollback aprobado, artefacto probado, plan de comunicación y verificación posterior.

## Registro de evidencia

Conserve un registro por cambio productivo con: identidad del release (SHA, digests, SBOM, entorno, ticket); aprobaciones; resultados de pruebas; excepciones y vencimientos; pruebas de restore/failover/rotación/rollback/incidentes; y verificaciones posteriores de salud, métricas, alertas y auditoría.

## Mapeo de marcos

Vea la [matriz de evidencia de aseguramiento](./matriz-evidencia-aseguramiento.md) para evidencia candidata mapeada a ISO/IEC 27001:2022, SOC 2 y NIST CSF 2.0. No demuestra diseño, efectividad operativa ni certificación.
