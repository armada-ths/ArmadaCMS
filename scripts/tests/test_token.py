"""Credential caching never assumes a newly-issued or fixed-lifetime token."""
import json
import sys
import unittest
from datetime import datetime, timezone
from pathlib import Path
from unittest.mock import Mock

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from release import ApiError, Cloud, ReleaseError
from test_release import Clock, config


def credential(token="secret-token", expiry=600):
    value = datetime.fromtimestamp(expiry, timezone.utc).isoformat() if expiry is not None else ""
    return json.dumps({"credential": {"access_token": token, "token_expiry": value}})


class TokenTests(unittest.TestCase):
    def cloud(self, values=None):
        clock = Clock()
        run = Mock(side_effect=values) if values is not None else Mock(return_value=credential())
        api = Mock(return_value={"done": True})
        cloud = Cloud(config(), run=run, api=api, clock=clock, sleep=clock.sleep, wall_clock=clock)
        return cloud, clock, run, api

    def test_calls_and_operation_polling_reuse_token(self):
        cloud, _, run, api = self.cloud()
        cloud.call("GET", "https://run.googleapis.com/v2/test")
        cloud.operation({"name": "projects/test/locations/test/operations/test"})
        cloud.call("POST", "https://run.googleapis.com/v2/test", {})
        self.assertEqual(run.call_count, 1)
        self.assertEqual(api.call_count, 3)
        self.assertIn("--min-expiry=300", run.call_args.args[0])

    def test_refreshes_before_actual_expiry_with_margin(self):
        cloud, clock, run, api = self.cloud([credential(expiry=600), credential("replacement", 1200)])
        cloud.call("GET", "test")
        clock.sleep(539)
        cloud.call("GET", "test")
        self.assertEqual(run.call_count, 1)
        clock.sleep(1)
        cloud.call("GET", "test")
        self.assertEqual(run.call_count, 2)
        self.assertEqual(api.call_args.args[2], "replacement")

    def test_missing_expiry_never_reuses_token(self):
        cloud, _, run, _ = self.cloud([credential(expiry=None), credential(expiry=None)])
        cloud.call("GET", "test")
        cloud.call("GET", "test")
        self.assertEqual(run.call_count, 2)

    def test_authentication_rejection_refreshes_once(self):
        cloud, _, run, api = self.cloud([credential(), credential("replacement"), credential("third")])
        api.side_effect = [ApiError(401), {}]
        cloud.call("PATCH", "test", {"etag": "original"})
        self.assertEqual(run.call_count, 2)
        self.assertIn("--force-auth-refresh", run.call_args.args[0])
        self.assertEqual(api.call_args.args[2:], ("replacement", {"etag": "original"}))
        api.side_effect = ApiError(401)
        with self.assertRaises(ApiError):
            cloud.call("GET", "test")
        self.assertEqual(api.call_count, 4)

    def test_permission_and_ambiguous_errors_are_not_retried(self):
        for status in (403, 409, 412, 500):
            cloud, _, run, api = self.cloud()
            api.side_effect = ApiError(status)
            with self.subTest(status=status), self.assertRaises(ApiError):
                cloud.call("PATCH", "test", {})
            self.assertEqual(api.call_count, 1)
            self.assertEqual(run.call_count, 1)

    def test_invalid_expired_or_unzoned_credentials_fail_without_leaking(self):
        for raw in ("secret-invalid-json", "{}", credential(expiry=30),
                    json.dumps({"credential": {"access_token": "secret", "token_expiry": "2026-10-18T12:00:00"}})):
            cloud, _, _, api = self.cloud([raw])
            with self.subTest(raw=raw), self.assertRaisesRegex(ReleaseError, "credential or expiry is invalid") as failure:
                cloud.call("GET", "test")
            self.assertNotIn("secret", str(failure.exception))
            api.assert_not_called()

    def test_refresh_failure_never_falls_back_to_old_token(self):
        cloud, clock, _, api = self.cloud([credential(), ReleaseError("Credential refresh failed")])
        cloud.call("GET", "test")
        clock.sleep(600)
        with self.assertRaises(ReleaseError):
            cloud.call("GET", "test")
        self.assertIsNone(cloud._token)
        self.assertEqual(api.call_count, 1)
