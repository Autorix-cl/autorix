import unittest
import httpx
from autorix import AutorixClient, AutorixHTTPError


def client(handler):
    return AutorixClient(nexus_url="https://nexus.test", themis_url="https://themis.test", janus_url="https://janus.test", vulcan_url="https://vulcan.test", ego_url="https://ego.test", enable_cache=False, http_client=httpx.Client(transport=httpx.MockTransport(handler)))

class ClientTests(unittest.TestCase):
    def test_check_sends_runtime_contract(self):
        def handler(request):
            self.assertEqual(request.url.path, "/check")
            self.assertEqual(request.json() if hasattr(request, 'json') else __import__('json').loads(request.content), {"namespace":"document","object":"1","relation":"view","subject_id":"u1","subject_namespace":"user","subject_relation":"","request_context":{},"explain":False})
            return httpx.Response(200, json={"allowed": True, "reason":"direct", "snap_token":"z1"})
        result = client(handler).check("document", "1", "view", "u1")
        self.assertTrue(result.allowed); self.assertEqual(result.snap_token, "z1")

    def test_policy_and_nexus_queries(self):
        def handler(request):
            if request.url.path == "/lookup/resources": return httpx.Response(200, json={"resources":["a"]})
            if request.url.path == "/policies/validate": return httpx.Response(200, json={"valid":True})
            return httpx.Response(404, json={"error":"not found"})
        c = client(handler)
        self.assertEqual(c.lookup_resources("doc", "view", "u1"), ["a"])
        self.assertTrue(c.validate_policy("true")["valid"])

    def test_http_errors_are_typed(self):
        with self.assertRaises(AutorixHTTPError) as error:
            client(lambda _: httpx.Response(400, json={"error":"bad input"})).expand("d", "1", "view")
        self.assertEqual(error.exception.status_code, 400)

    def test_oauth_authorization_url_requires_pkce_inputs(self):
        url = client(lambda _: httpx.Response(500)).authorization_url("public", "https://app.test/cb", ["openid"], "state", "challenge")
        self.assertIn("code_challenge=challenge", url); self.assertIn("code_challenge_method=S256", url)

    def test_vulcan_rejects_legacy_string_token_without_network_call(self):
        result = client(lambda _: self.fail("unexpected request")).verify_api_key("raw-token")
        self.assertFalse(result.valid); self.assertIn("macaroon", result.error)
