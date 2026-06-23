"""Unit tests for the provider key rotators. No network: request_json is stubbed."""

import unittest
from unittest import mock

import rotate_provider_key as r


class FakeHTTP:
    """Records (method, url) calls and replies from a scripted routing table."""

    def __init__(self, routes):
        self.routes = routes
        self.calls = []

    def __call__(self, method, url, headers, payload=None, timeout=30):
        self.calls.append((method, url, payload))
        for (m, needle), response in self.routes.items():
            if m == method and needle in url:
                return response(url, payload) if callable(response) else response
        raise AssertionError(f"unexpected request {method} {url}")


class ByoKeyTests(unittest.TestCase):
    def test_mistral_resolves_and_validates_new_key(self):
        http = FakeHTTP({("GET", "api.mistral.ai/v1/models"): {"data": [{"id": "mistral-large"}]}})
        with mock.patch.object(r, "request_json", http), mock.patch.dict("os.environ", {}, clear=True):
            result = r.create_mistral_key("alias", {"MISTRAL_NEW_API_KEY": "newkey"})
        self.assertEqual(result.env_key, "MISTRAL_API_KEY")
        self.assertEqual(result.secret, "newkey")
        self.assertEqual(result.metadata["new_key_source"], "MISTRAL_NEW_API_KEY")
        self.assertEqual(http.calls[0][0], "GET")

    def test_nous_missing_new_key_raises(self):
        with mock.patch.dict("os.environ", {}, clear=True):
            with self.assertRaises(r.RotationError):
                r.create_nous_key("alias", {})

    def test_byo_rejects_non_list_models(self):
        http = FakeHTTP({("GET", "api.mistral.ai/v1/models"): {"data": "nope"}})
        with mock.patch.object(r, "request_json", http), mock.patch.dict("os.environ", {}, clear=True):
            with self.assertRaises(r.RotationError):
                r.create_mistral_key("alias", {"MISTRAL_NEW_API_KEY": "newkey"})


class OpenRouterTests(unittest.TestCase):
    def test_create_returns_secret_and_hash(self):
        http = FakeHTTP({
            ("POST", "openrouter.ai/api/v1/keys"): {"key": "sk-or-v1-new", "data": {"hash": "h1", "name": "n"}},
            ("GET", "openrouter.ai/api/v1/key"): {"data": {"label": "sk-or-v1-...new"}},
        })
        with mock.patch.object(r, "request_json", http):
            result = r.create_openrouter_key("alias", {"OPENROUTER_PROVISIONING_KEY": "prov"})
        self.assertEqual(result.secret, "sk-or-v1-new")
        self.assertEqual(result.metadata["hash"], "h1")

    def test_revoke_matches_unique_redacted_label(self):
        def list_page(url, _payload):
            if "offset=0" in url:
                return {"data": [
                    {"hash": "old", "label": "sk-or-v1-...beef"},
                    {"hash": "newh", "label": "sk-or-v1-...cafe"},
                ]}
            return {"data": []}  # next page empty -> pagination terminates

        http = FakeHTTP({
            ("GET", "openrouter.ai/api/v1/keys"): list_page,
            ("DELETE", "openrouter.ai/api/v1/keys/old"): {},
        })
        with mock.patch.object(r, "request_json", http):
            out = r.revoke_openrouter_key("prov", "sk-or-v1-secretbeef", new_hash="newh")
        self.assertTrue(out["revoked"])
        self.assertEqual(out["hash"], "old")
        self.assertIn(("DELETE", "https://openrouter.ai/api/v1/keys/old", None), http.calls)

    def test_revoke_skips_when_ambiguous(self):
        http = FakeHTTP({("GET", "openrouter.ai/api/v1/keys"): {"data": []}})
        with mock.patch.object(r, "request_json", http):
            out = r.revoke_openrouter_key("prov", "", new_hash="newh")
        self.assertFalse(out["revoked"])


class XaiTests(unittest.TestCase):
    def test_team_id_discovered_from_current_key(self):
        http = FakeHTTP({("GET", "api.x.ai/v1/api-key"): {"team_id": "team-123", "api_key_id": "k1"}})
        with mock.patch.object(r, "request_json", http), mock.patch.dict("os.environ", {}, clear=True):
            team = r.resolve_xai_team_id({"XAI_API_KEY": "xai-current"})
        self.assertEqual(team, "team-123")

    def test_create_uses_team_and_validates(self):
        http = FakeHTTP({
            ("GET", "api.x.ai/v1/api-key"): {"team_id": "team-123", "api_key_id": "kid"},
            ("POST", "management-api.x.ai/auth/teams/team-123/api-keys"): {"apiKey": "xai-new", "apiKeyId": "new-id"},
        })
        with mock.patch.object(r, "request_json", http), mock.patch.dict("os.environ", {}, clear=True):
            result = r.create_xai_key("alias", {"XAI_MANAGEMENT_KEY": "mgmt", "XAI_API_KEY": "xai-current"})
        self.assertEqual(result.secret, "xai-new")
        self.assertEqual(result.metadata["api_key_id"], "new-id")
        self.assertEqual(result.metadata["team_id"], "team-123")

    def test_revoke_deletes_by_discovered_id(self):
        http = FakeHTTP({
            ("GET", "api.x.ai/v1/api-key"): {"api_key_id": "old-id"},
            ("DELETE", "management-api.x.ai/auth/api-keys/old-id"): {},
        })
        with mock.patch.object(r, "request_json", http):
            out = r.revoke_xai_key("mgmt", "xai-old")
        self.assertTrue(out["revoked"])
        self.assertEqual(out["api_key_id"], "old-id")


if __name__ == "__main__":
    unittest.main()
