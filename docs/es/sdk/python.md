# SDK Oficial de Python para Autorix

El SDK oficial de Python (`autorix`) está optimizado para **FastAPI**, **Django** y **Flask** con soporte síncrono y asíncrono (`asyncio` / `httpx`).

---

## 📦 Instalación

```bash
pip install autorix
```

---

## 🚀 Uso Rápido con FastAPI

```python
from fastapi import FastAPI, Depends, HTTPException
from autorix import AutorixClient

app = FastAPI()
client = AutorixClient(
    nexus_url="http://localhost:8080",
    themis_url="http://localhost:4488"
)

@app.get("/documents/{doc_id}")
async def get_document(doc_id: str, user_id: str):
    res = await client.nexus.check_async(
        namespace="documents",
        object=doc_id,
        relation="viewer",
        subject=f"user:{user_id}"
    )
    if not res.allowed:
        raise HTTPException(status_code=403, detail="Acceso denegado")
    return {"doc_id": doc_id, "content": "Contenido confidencial"}
```
