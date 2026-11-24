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

    def test_verify_api_key_populates_fields_with_keyword_args(self):
        def handler(request):
            self.assertEqual(request.url.path, "/keys/verify")
            return httpx.Response(200, json={
                "valid": True,
                "api_key": {"id": "key-123", "name": "ci-bot", "scopes": ["read", "write"]},
            })
        result = client(handler).verify_api_key({"location": "https://api.test", "key_id": "key-123", "caveats": [], "signature": "abc"})
        self.assertTrue(result.valid)
        self.assertIsNone(result.error)
        self.assertEqual(result.key_id, "key-123")
        self.assertEqual(result.name, "ci-bot")
        self.assertEqual(result.scopes, ["read", "write"])

    def test_write_and_delete_tuples_wrap_payload(self):
        seen = []
        def handler(request):
            seen.append((request.method, __import__("json").loads(request.content)))
            return httpx.Response(201 if request.method == "POST" else 200, json=[])
        c = client(handler)
        t = {"namespace": "doc", "object": "1", "relation": "viewer", "subject_id": "alice"}
        c.write_tuples([t])
        c.delete_tuples([t])
        self.assertEqual(seen[0], ("POST", {"tuples": [t]}))
        self.assertEqual(seen[1], ("DELETE", {"tuples": [t]}))

    def test_dynamic_path_params_are_url_escaped(self):
        seen_paths = []
        def handler(request):
            seen_paths.append(request.url.raw_path.decode("ascii"))
            return httpx.Response(200, json={})
        c = client(handler)
        c.get_policy("../admin/secret")
        c.revoke_api_key("k/1?x=2")
        self.assertEqual(seen_paths[0].split("?")[0], "/policies/..%2Fadmin%2Fsecret")
        self.assertEqual(seen_paths[1], "/keys/k%2F1%3Fx%3D2")

    def test_cache_bypass_for_context_snap_token_explain_and_bounded(self):
        calls = 0
        def handler(request):
            nonlocal calls
            calls += 1
            return httpx.Response(200, json={"allowed": True, "reason": "ok", "trace": {"n": calls}})
        c = AutorixClient(
            nexus_url="https://nexus.test",
            enable_cache=True,
            max_cache_size=2,
            http_client=httpx.Client(transport=httpx.MockTransport(handler)),
        )
        c.check("doc", "1", "view", "u1")
        c.check("doc", "1", "view", "u1")  # cache hit
        self.assertEqual(calls, 1)
        c.check("doc", "1", "view", "u1", context={"ip": "1.2.3.4"})  # bypass
        c.check("doc", "1", "view", "u1", snap_token="zk1")  # bypass
        c.check("doc", "1", "view", "u1", explain=True)  # bypass
        c.check("doc", "1", "view", "u1", subject_relation="member")  # distinct key
        self.assertEqual(calls, 5)
        c.check("doc", "2", "view", "u2")  # fills max_cache_size=2, evicts oldest
        self.assertLessEqual(len(c._cache), 2)

    def test_public_pkce_token_does_not_leak_api_key_and_wraps_http_error(self):
        def handler(request):
            self.assertNotIn("authorization", {k.lower() for k in request.headers.keys()})
            return httpx.Response(200, json={"access_token": "tok", "token_type": "Bearer", "expires_in": 60})
        c = AutorixClient(
            janus_url="https://janus.test",
            api_key="av_live_super_secret",
            http_client=httpx.Client(transport=httpx.MockTransport(handler)),
        )
        tok = c.exchange_code("spa", "code", "https://app.test/cb", "verifier")
        self.assertEqual(tok.access_token, "tok")

        def fail_handler(request):
            raise httpx.ConnectError("connection refused")
        c_fail = AutorixClient(
            janus_url="https://janus.test",
            http_client=httpx.Client(transport=httpx.MockTransport(fail_handler)),
        )
        from autorix import AutorixError
        with self.assertRaises(AutorixError):
            c_fail.exchange_code("spa", "code", "https://app.test/cb", "verifier")
