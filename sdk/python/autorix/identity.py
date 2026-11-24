"""Cryptographic verification for Aegis-injected identity headers."""
from __future__ import annotations

import hashlib
import hmac
import os
from typing import Optional

IDENTITY_SIGNATURE_HEADER = "X-Autorix-Identity-Signature"


def default_identity_signing_secret() -> Optional[str]:
    secret = os.getenv("AUTORIX_IDENTITY_SIGNING_SECRET") or os.getenv("AEGIS_IDENTITY_SIGNING_SECRET")
    return secret.strip() if secret and secret.strip() else None


def sign_identity_headers(
    user_id: Optional[str],
    email: Optional[str] = None,
    roles: Optional[str] = None,
    scopes: Optional[str] = None,
    secret: str = "",
) -> str:
    canonical = f"v1\n{user_id or ''}\n{email or ''}\n{roles or ''}\n{scopes or ''}"
    digest = hmac.new(secret.encode("utf-8"), canonical.encode("utf-8"), hashlib.sha256).hexdigest()
    return f"v1={digest}"


def verify_identity_headers(
    user_id: Optional[str],
    email: Optional[str] = None,
    roles: Optional[str] = None,
    scopes: Optional[str] = None,
    signature: Optional[str] = None,
    secret: Optional[str] = None,
) -> bool:
    if not secret or not secret.strip() or not signature or not signature.strip():
        return False
    expected = sign_identity_headers(user_id, email, roles, scopes, secret.strip())
    return hmac.compare_digest(signature.strip(), expected)
