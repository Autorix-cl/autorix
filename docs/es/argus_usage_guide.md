# Autorix Argus: Manual del Plano de Control y Auditoría Criptográfica

**Autorix Argus** es el plano de control central y registro de flota de la suite Autorix. Gestiona el registro dinámico de motores mediante tokens de enrolamiento (`aet_`), recolecta heartbeats de salud, mantiene un registro inmutable de auditoría encadenado con **SHA-256 Merkle Hash-Chain** y genera evidencias de cumplimiento para **SOC 2** e **ISO 27001**.

---

## 🛡️ 1. Cadena de Auditoría Inmutable

Cada mutación en el clúster genera un registro encadenado criptográficamente:
```text
Record Hash = SHA-256(prev_hash + id + action + resource_type + resource_id + outcome + created_at)
```

### Verificación de Integridad (`GET /v1/audit/verify`)
Verifica matemáticamente que la cadena histórica no haya sido alterada:

```bash
curl -X GET http://localhost:4400/v1/audit/verify \
  -H "Authorization: Bearer ast_..."
```

---

## 📡 2. Referencia de la API REST

Argus corre en el puerto HTTP `4400` y gRPC `50053`.

- `POST /v1/auth/bootstrap`: Reclama la cuenta inicial de Master Owner con el token `abt_...`.
- `POST /v1/enrollment-tokens`: Genera un token `aet_...` de un solo uso para enrolar un motor.
- `GET /v1/instances`: Lista todas las instancias activas de la flota y su estado de salud.
- `GET /v1/audit`: Consulta los registros de auditoría estructurados.
- `GET /v1/compliance/report`: Descarga el reporte continuo de cumplimiento.
