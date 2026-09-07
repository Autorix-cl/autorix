# Inicio con Docker Compose

Docker Compose es solo para **desarrollo y evaluación local**. Usa puertos
locales, credenciales de desarrollo, HTTP entre servicios y `sslmode=disable`.
No debe exponerse a Internet ni tratarse como un despliegue productivo.

## Iniciar el stack

Requisitos: Docker Engine con Compose v2, 8 GB de RAM y los puertos `3000`,
`4444`, `4455` y `5432` disponibles.

```bash
./scripts/generate-local-secrets.sh
docker compose --profile core up -d --build
docker compose --profile core ps
```

El generador debe ejecutarse antes del comando Compose en cada instalación
local nueva. Crea secretos de desarrollo no versionados; su salida no debe
incluirse en el control de versiones.

Cuando los servicios estén saludables, Console estará disponible en
`http://localhost:3000`. Obtenga el token único de bootstrap de Argus desde los
logs y complete el flujo de instalación:

```bash
docker logs autorix-argus | grep -i 'bootstrap token'
```

No copies tokens de bootstrap ni contraseñas a control de versiones, historial
de shell o tickets compartidos.

```bash
curl -fsS http://localhost:4444/health/ready
curl -fsS http://localhost:4455/health/ready
curl -fsS http://localhost:4400/health/ready
```

La administración de Janus (`:4445`) y Aegis (`:4456`) no está publicada por
Compose. Utilice Console o un acceso local autenticado; no agregue puertos de
host para esos listeners.
