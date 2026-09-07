import unittest
from autorix.fastapi import AutorixSecurity
from autorix.flask import current_user as flask_current_user
from autorix.django import get_current_user

class IntegrationsTests(unittest.TestCase):
    def test_security_parses_trusted_identity_headers(self):
        user = AutorixSecurity().get_current_user("u1", "u@example.test", "admin, editor")
        self.assertEqual(user.roles, ["admin", "editor"])
        self.assertIsNone(AutorixSecurity().get_current_user())

    def test_optional_framework_modules_import(self):
        # Modules do not import optional frameworks until their adapter is used.
        self.assertTrue(callable(flask_current_user)); self.assertTrue(callable(get_current_user))
