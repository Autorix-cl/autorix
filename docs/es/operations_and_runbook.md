# Runbook de Operaciones: Día 1 y Día 2

Este runbook describe las tareas operativas esenciales para el despliegue, mantenimiento, respaldo y recuperación ante desastres de un clúster **Autorix**.

---

## 1. Inicialización y Bootstrap del Clúster

### Obtener el Token de Bootstrap Inicial
Cuando Argus arranca por primera vez en un entorno no inicializado, genera un token con prefijo `abt_...`:

```bash
docker logs autorix-argus | grep "BOOTSTRAP TOKEN"
```

Ingresa a `http://localhost:3000/setup` o ejecuta:

```bash
curl -X POST http://localhost:4400/v1/auth/bootstrap \
    -H "Content-Type: application/json" \
    -d '{
        "bootstrap_token": "abt_1c60ad03bb589af2ceea1439fe08d0fe7f82c16527a6981b73a36072439d8419",
        "email": "admin@empresa.corp",
        "password": "PasswordSuperSeguro#2026",
        "name": "Administrador Principal"
    }'
```

---

## 2. Respaldo y Restauración de Bases de Datos

Cada motor utiliza su propia base de datos aislada en PostgreSQL:

```bash
DATABASES=("autorix_argus" "autorix_nexus" "autorix_ego" "autorix_janus" "autorix_vulcan" "autorix_hermes" "autorix_themis" "autorix_aegis")

for DB in "${DATABASES[@]}"; do
  pg_dump -h localhost -U autorix -d "$DB" -F c -b -v -f "${DB}_backup_$(date +%Y%m%d).dump"
done
```
