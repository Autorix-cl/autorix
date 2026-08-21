# CLI Oficial de Autorix (`autorixctl`) y APIs Directas

`autorixctl` es la herramienta de línea de comandos para automatizar tareas administrativas, gestionar tuplas de permisos y enrolar motores en clústeres de Autorix.

---

## 📦 Instalación

```bash
# Vía Go
go install github.com/autorix-cl/autorix/cmd/autorixctl@latest

# O descarga el binario desde GitHub Releases
curl -fsSL https://get.autorix.io | sh
```

---

## 🛠️ Comandos Comunes

```bash
# Verificar permisos
autorixctl check documents roadmap_2026 viewer user:alice

# Insertar tupla de relación
autorixctl tuple add documents roadmap_2026 editor user:bob

# Enrolar un nuevo motor
autorixctl fleet enroll --token aet_01917f8a7b6c...
```
