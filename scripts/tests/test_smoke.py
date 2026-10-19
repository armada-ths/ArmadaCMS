"""Check HTTP identity, API pagination and real admin asset responses."""
import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from collections.abc import Callable

from scripts.smoke_test import SmokeError, smoke
from scripts.tests.test_release import Clock

COMMIT = "a" * 40


HttpResponse = tuple[int, dict[str, str], bytes]


class SmokeHTTPServer(ThreadingHTTPServer):
    """Local server with a replaceable, typed response callback."""
    responses: Callable[[str], HttpResponse]


class SmokeTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_args: object, **_kwargs: object) -> None:
                pass

            def do_GET(self):
                server = self.server
                assert isinstance(server, SmokeHTTPServer)
                status, headers, body = server.responses(self.path)
                self.send_response(status)
                for key, value in headers.items():
                    self.send_header(key, value)
                self.end_headers()
                self.wfile.write(body)
        cls.server = SmokeHTTPServer(("127.0.0.1", 0), Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.url = "http://127.0.0.1:" + str(cls.server.server_port)

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def setUp(self):
        self.clock = Clock()
        self.server.responses = self.response

    @staticmethod
    def response(path):
        if path == "/health":
            return 200, {"X-Armada-Commit": COMMIT, "X-Armada-Revision": "revision-new", "Cache-Control": "no-store"}, b"OK"
        if path.startswith("/api/v1/featureflags"):
            return 200, {"Content-Type": "application/json", "Content-Range": "featureflags 0-0/1"}, json.dumps([{"key": "MAP_PAGE", "enabled": True}]).encode()
        if path == "/admin/":
            return 200, {"Content-Type": "text/html"}, b'<html><div id="root"></div><script src="/admin/assets/app.js"></script><link rel="stylesheet" href="./assets/app.css"></html>'
        if path.endswith(".js"):
            return 200, {"Content-Type": "text/javascript"}, b"console.log('app')"
        if path.endswith(".css"):
            return 200, {"Content-Type": "text/css"}, b"body{margin:0}"
        return 404, {}, b"missing"

    def check(self, **kwargs):
        smoke(self.url, timeout=1, clock=self.clock, sleep=self.clock.sleep, **kwargs)

    def test_revision_and_database_backed_admin_checks_over_real_http(self):
        self.check(commit=COMMIT, revision="revision-new")

    def test_wrong_commit_or_revision_is_rejected(self):
        for kwargs in ({"commit": "b" * 40}, {"revision": "old"}):
            self.clock = Clock()
            with self.subTest(kwargs=kwargs), self.assertRaises(SmokeError):
                self.check(**kwargs)

    def test_legacy_rollback_can_omit_identity_headers(self):
        def response(path):
            status, headers, body = self.response(path)
            if path == "/health":
                headers = {}
            return status, headers, body
        self.server.responses = response
        self.check()
        with self.assertRaises(SmokeError):
            self.check(commit=COMMIT)

    def test_http_failure_is_rejected(self):
        self.server.responses = lambda path: (503, {}, b"not ready")
        with self.assertRaises(SmokeError):
            self.check()

    def test_missing_headers_invalid_json_and_pagination_are_rejected(self):
        for defect in ("range", "json", "pagination", "content-type"):
            def response(path, current_defect=defect):
                status, headers, body = self.response(path)
                if path.startswith("/api/v1/"):
                    if current_defect == "range":
                        headers.pop("Content-Range")
                    elif current_defect == "json":
                        body = b"not-json"
                    elif current_defect == "pagination":
                        body = b"[{},{}]"
                    else:
                        headers["Content-Type"] = "text/html"
                return status, headers, body
            self.clock = Clock()
            self.server.responses = response
            with self.subTest(defect=defect), self.assertRaises(SmokeError):
                self.check()

    def test_missing_asset_and_spa_fallback_are_rejected(self):
        for defect in ("missing", "fallback", "empty"):
            def response(path, current_defect=defect):
                if path.endswith(".js"):
                    if current_defect == "missing":
                        return 404, {}, b"missing"
                    if current_defect == "empty":
                        return 200, {"Content-Type": "text/javascript"}, b""
                    return self.response("/admin/")
                return self.response(path)
            self.clock = Clock()
            self.server.responses = response
            with self.subTest(defect=defect), self.assertRaises(SmokeError):
                self.check()

    def test_one_deadline_covers_all_requests_and_retries(self):
        calls = []
        def get(_url, timeout):
            calls.append(timeout)
            self.clock.sleep(0.6)
            raise OSError("network unavailable")
        with self.assertRaises(SmokeError):
            smoke(self.url, timeout=1, get=get, clock=self.clock, sleep=self.clock.sleep)
        self.assertEqual(calls, [1])
        self.assertEqual(self.clock.now, 1)

    def test_cross_origin_assets_are_rejected_without_request(self):
        def response(path):
            if path == "/admin/":
                return 200, {"Content-Type": "text/html"}, b'<div id="root"></div><script src="https://other.invalid/a.js"></script><link rel="stylesheet" href="/app.css">'
            return self.response(path)
        self.server.responses = response
        with self.assertRaisesRegex(SmokeError, "not served by this CMS"):
            self.check()


    def test_google_fonts_does_not_replace_local_css_or_require_external_fetch(self):
        calls = []
        def response(path):
            calls.append(path)
            status, headers, body = self.response(path)
            if path == "/admin/":
                body += b'<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Roboto">'
            return status, headers, body
        self.server.responses = response
        self.check(commit=COMMIT, revision="revision-new")
        self.assertIn("/admin/assets/app.css", calls)
        self.assertFalse(any("fonts.googleapis.com" in path for path in calls))

    def test_google_fonts_alone_cannot_satisfy_css_entrypoint_check(self):
        def response(path):
            status, headers, body = self.response(path)
            if path == "/admin/":
                body = b'<div id="root"></div><script src="/app.js"></script><link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Roboto">'
            return status, headers, body
        self.server.responses = response
        with self.assertRaisesRegex(SmokeError, "entrypoint assets are missing"):
            self.check()

    def test_unexpected_external_stylesheet_is_rejected(self):
        for url in ("https://other.invalid/app.css", "https://fonts.googleapis.com.evil.invalid/css2"):
            def response(path, stylesheet_url=url):
                status, headers, body = self.response(path)
                if path == "/admin/":
                    body += f'<link rel="stylesheet" href="{stylesheet_url}">'.encode()
                return status, headers, body
            self.clock = Clock()
            self.server.responses = response
            with self.subTest(url=url), self.assertRaisesRegex(SmokeError, "not served by this CMS"):
                self.check()

    def test_identified_revision_must_disable_health_caching(self):
        def response(path):
            status, headers, body = self.response(path)
            if path == "/health":
                headers.pop("Cache-Control")
            return status, headers, body
        self.server.responses = response
        with self.assertRaises(SmokeError):
            self.check(commit=COMMIT)

    def test_recovery_checks_available_revision_headers_and_accepts_legacy(self):
        self.check(accepted_revisions=("revision-new",))
        with self.assertRaises(SmokeError):
            self.check(accepted_revisions=("old",))
        self.clock = Clock()
        def response(path):
            status, headers, body = self.response(path)
            if path == "/health":
                headers = {}
            return status, headers, body
        self.server.responses = response
        self.check(accepted_revisions=("old",))


if __name__ == "__main__":
    unittest.main()
