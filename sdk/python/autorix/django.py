"""Django integration; identity headers must be set by a trusted proxy."""
from functools import wraps
from .client import AutorixClient, User

def get_current_user(request):
    value = request.headers.get("X-User-ID")
    return User(value, request.headers.get("X-User-Email"), [r.strip() for r in request.headers.get("X-User-Roles", "").split(",") if r.strip()]) if value else None

def require_permission(client: AutorixClient, namespace: str, relation: str, object_id):
    def decorator(view):
        @wraps(view)
        def wrapped(request, *args, **kwargs):
            from django.http import HttpResponse, HttpResponseForbidden
            user = get_current_user(request)
            if user is None: return HttpResponse("Authentication required", status=401)
            target = object_id(request, *args, **kwargs) if callable(object_id) else object_id
            if not target or not client.check(namespace, target, relation, user.id).allowed: return HttpResponseForbidden("Permission denied")
            return view(request, *args, **kwargs)
        return wrapped
    return decorator
