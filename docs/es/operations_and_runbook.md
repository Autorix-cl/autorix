# Runbook de Operaciones: Día 1 y Día 2

Este runbook describe las tareas operativas esenciales para el despliegue, mantenimiento, respaldo y recuperación ante desastres de un clúster **Autorix**.

---

## 1. Inicialización y Bootstrap del Clúster

### Obtener el Token de Bootstrap Inicial
Cuando Argus arranca por primera vez en un entorno no inicializado, genera un token con prefijo `abt_...`:

```bash
docker logs autorix-argus | grep "BOOTSTRAP TOKEN"
```

Acceda a `http://localhost:3000/setup` o ejecute:

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

---

## Alertmanager opcional (autohospedado)

Prometheus evalúa el archivo de reglas incluido. La entrega de notificaciones
permanece **deshabilitada** hasta que se configure deliberadamente Alertmanager
con un receptor administrado por el operador. Este repositorio no incluye URL
de webhook, credenciales SMTP ni tokens de paginación.

### Docker Compose

1. Copie `deploy/monitoring/alertmanager.example.yml` fuera del repositorio y
   sustituya el receptor `discard` por una configuración revisada de webhook,
   correo electrónico o paginación. Mantenga las credenciales en un gestor de
   secretos o en un archivo ignorado por Git.
2. Inicie el monitoreo y las notificaciones juntos:
   ```bash
   ALERTMANAGER_CONFIG_PATH=/secure/path/alertmanager.yml \
     docker compose --profile core --profile extended --profile alerts up -d prometheus alertmanager
   ```
3. Compruebe que Alertmanager esté disponible solo en el host local en
   `http://127.0.0.1:9093`. Prometheus continúa siendo utilizable cuando el
   perfil `alerts` no está habilitado, pero no puede entregar notificaciones.

### Helm para Kubernetes

Cree el Secret de configuración fuera de los valores de Helm. La clave
`alertmanager.yml` debe contener la configuración completa de Alertmanager:

```bash
kubectl -n autorix create secret generic autorix-alertmanager-config \
  --from-file=alertmanager.yml=/secure/path/alertmanager.yml
```

Después, proporcione un archivo de valores con imagen fijada por digest:

```yaml
monitoring:
  alertmanager:
    enabled: true
    existingConfigSecret: autorix-alertmanager-config
    configKey: alertmanager.yml
    image:
      digest: sha256:<verified-alertmanager-image-digest>
```

El chart monta ese Secret existente en modo de solo lectura y utiliza estado de
alertas local efímero. Configure estado durable y alta disponibilidad mediante
un operador de Alertmanager dedicado si sus objetivos de recuperación lo
requieren.
