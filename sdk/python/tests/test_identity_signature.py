import unittest

from autorix import AutorixClient, sign_identity_headers, verify_identity_headers
from autorix.fastapi import AutorixSecurity


class IdentitySignatureTests(unittest.TestCase):
    def test_sign_and_verify_roundtrip(self):
        secret = "shared-proxy-secret"
        sig = sign_identity_headers("usr_42", "alice@autorix.io", "admin,viewer", "read write", secret)
        self.assertTrue(sig.startswith("v1="))
        self.assertTrue(
            verify_identity_headers("usr_42", "alice@autorix.io", "admin,viewer", "read write", sig, secret)
        )
        # Tampered subject must fail
        self.assertFalse(
            verify_identity_headers("usr_admin", "alice@autorix.io", "admin,viewer", "read write", sig, secret)
        )
        # Missing signature or secret must fail
        self.assertFalse(verify_identity_headers("usr_42", None, None, None, None, secret))
        self.assertFalse(verify_identity_headers("usr_42", None, None, None, sig, ""))

    def test_fastapi_security_enforces_signature(self):
        secret = "fastapi-secret"
        sec = AutorixSecurity(client=AutorixClient(), identity_signing_secret=secret)

        # Unsigned spoofed header rejected
        self.assertIsNone(sec.get_current_user("usr_spoofed", "a@b.com", "admin"))

        # Valid signed header accepted
        sig = sign_identity_headers("usr_42", "alice@autorix.io", "admin, viewer", None, secret)
        user = sec.get_current_user("usr_42", "alice@autorix.io", "admin, viewer", None, sig)
        self.assertIsNotNone(user)
        self.assertEqual(user.id, "usr_42")
        self.assertEqual(user.roles, ["admin", "viewer"])

        # Fail-closed require_signature without secret rejects
        fail_closed = AutorixSecurity(client=AutorixClient(), identity_signing_secret="", require_signature=True)
        self.assertIsNone(fail_closed.get_current_user("usr_42"))


if __name__ == "__main__":
    unittest.main()
