# Matriz de evidencia de aseguramiento

Esta matriz ayuda a los operadores a recopilar evidencia para sus propias evaluaciones ISO/IEC 27001:2022, SOC 2 y NIST CSF 2.0. No declara cumplimiento, certificación, atestación, auditoría completada ni efectividad operativa. Un evaluador calificado y la organización desplegadora determinan alcance, aplicabilidad y aceptación de riesgo.

## Uso de la matriz

1. Seleccione los controles aplicables a la organización y al despliegue.
2. Capture evidencia del repositorio y del ambiente para el mismo período de release.
3. Pruebe la efectividad operativa, registre excepciones y haga que responsable y evaluador valoren el resultado.

La evidencia de repositorio solo indica un punto de partida de implementación. La evidencia operativa sigue siendo responsabilidad del operador.

| Objetivo | ISO/IEC 27001:2022 | SOC 2 | NIST CSF 2.0 | Evidencia de repositorio candidata | Evidencia operativa requerida | Estado |
|---|---|---|---|---|---|---|
| Control de acceso y mínimo privilegio | A 5.15, 5.18, 8.2, 8.3 | CC6.1–CC6.3 | Protect (PR.AA) | OAuth/OIDC, RBAC, ReBAC/ABAC, límites admin privados y pruebas | Ciclo de identidad, MFA, revisiones de acceso y aprobaciones privilegiadas | Parcial |
| Autenticación y credenciales | A 5.17, 8.5 | CC6.1, CC6.6 | Protect (PR.AA, PR.DS) | Ego, flujos Janus y validación JWT Aegis | Políticas contraseña/MFA, inventario de secretos, custodia de claves y excepciones | Parcial |
| Configuración segura | A 8.9, 8.20, 8.22 | CC6.6, CC7.1 | Protect (PR.PS, PR.IR) | Contextos Helm, NetworkPolicy, listeners privados y guards | Admisión de clúster, baselines, revisiones, prueba egress y excepciones | Parcial |
| Gestión de claves | A 8.24 | CC6.1, CC6.7 | Protect (PR.DS) | Claves Janus persistentes, JWKS, rotación y validación | HSM/KMS, separación de deberes, ceremonias, escrow y revocación | Parcial |
| Logs, monitoreo y auditoría | A 8.15–8.17 | CC7.1, CC7.2 | Detect (DE.CM), Govern (GV.OC) | Auditoría Argus, métricas y runbook | Integridad de logs centralizados, SIEM, retención, sincronía temporal y alertas probadas | Parcial |
| Vulnerabilidades y cambios | A 8.8, 8.32 | CC7.1, CC8.1 | Identify (ID.RA), Govern (GV.RM) | CI, historial, `SECURITY.md` y checklist | SBOM/provenance, scans, SLA, CAB y revisión independiente | Del operador |
| Respuesta a incidentes | A 5.24–5.28 | CC7.3, CC7.4 | Respond (RS.MA, RS.CO) | Política de reporte y runbook | Plan, on-call, tabletop, preservación de evidencia y notificaciones | Del operador |
| Backup y recuperación | A 8.13, 5.30 | A1.2, CC7.5 | Recover (RC.RP) | Runbook y ejemplos backup/restore | RTO/RPO aprobado, backups cifrados, drills restore/failover y continuidad | Del operador |
| Riesgo de proveedores | A 5.19–5.23 | CC9.2 | Govern (GV.SC) | Manifiestos de dependencias y referencias de despliegue | Inventario, evaluación de riesgo, contratos y monitoreo | Del operador |
| Privacidad y regulación | Dependiente del contexto | Dependiente del contexto | Govern (GV.OC, GV.RM) | Flujos de datos del producto cuando existan | Base legal, DPIA, retención/eliminación, residencia y evaluaciones regulatorias | Del operador |

## Calidad de la evidencia

La evidencia debe estar vinculada a un release y responsable, tener fecha, protección contra alteración, retención según política y revisión independiente. Capturas solas son débiles: combínelas con logs inmutables, artefactos firmados, exports de configuración, resultados de pruebas, tickets y aprobaciones.

Una exportación de auditoría o dashboard es material de evidencia, no prueba por sí sola de que un control funcionó durante un período. La organización debe validar integridad, retención, acceso y el proceso que la generó.

## No afirmaciones

Autorix no afirma certificación ISO 27001, atestación SOC 2, conformidad NIST CSF, cumplimiento PCI DSS ni aprobación para banca. Código, configuración, controles cloud, personal y alcance regulatorio varían por despliegue y no pueden certificarse desde un repositorio.
