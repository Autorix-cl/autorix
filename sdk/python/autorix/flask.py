"""Flask integration; identity headers must be set by a trusted proxy."""
from functools import wraps
from .client import AutorixClient, User

def require_permission(client: AutorixClient, namespace: str, relation: str, object_id=lambda: None):
    def decorator(view):
        @wraps(view)
        def wrapped(*args, **kwargs):
            from flask import abort, request
            user_id = request.headers.get("X-User-ID")
            if not user_id: abort(401)
            target = object_id() if callable(object_id) else object_id
            if not target or not client.check(namespace, target, relation, user_id).allowed: abort(403)
            return view(*args, **kwargs)
        return wrapped
    return decorator

def current_user():
    from flask import request
    value = request.headers.get("X-User-ID")
    return User(value, request.headers.get("X-User-Email"), [r.strip() for r in request.headers.get("X-User-Roles", "").split(",") if r.strip()]) if value else None
