"""Read-only HTTP verification for a deployed CMS revision."""

import argparse
import json
import re
import time
from html.parser import HTMLParser
from urllib.error import HTTPError
from urllib.parse import urljoin, urlsplit
from urllib.request import Request, urlopen


class SmokeError(RuntimeError):
    pass


class AdminAssets(HTMLParser):
    def __init__(self):
        super().__init__()
        self.scripts = []
        self.styles = []

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == "script" and attrs.get("src"):
            self.scripts.append(attrs["src"])
        if tag == "link" and attrs.get("rel") == "stylesheet" and attrs.get("href"):
            self.styles.append(attrs["href"])


def fetch(url, timeout):
    request = Request(url, headers={"Cache-Control": "no-cache"})
    try:
        response = urlopen(request, timeout=timeout)
    except HTTPError as error:
        status = error.code
        error.close()
        raise SmokeError("Unexpected HTTP status " + str(status)) from None
    with response:
        if response.status != 200:
            raise SmokeError("Unexpected HTTP status")
        if (urlsplit(response.url).netloc, urlsplit(response.url).scheme) != (urlsplit(url).netloc, urlsplit(url).scheme):
            raise SmokeError("Unexpected cross-origin redirect")
        body = response.read(5 * 1024 * 1024 + 1)
        if len(body) > 5 * 1024 * 1024:
            raise SmokeError("Response exceeds smoke-test limit")
        return response.headers, body


def smoke(base_url, commit=None, revision=None, *, timeout=120, get=fetch,
          clock=time.monotonic, sleep=time.sleep, accepted_revisions=None):
    """Retry the entire check within one deadline; do not modify hosted data."""
    base_url = base_url.rstrip("/") + "/"
    deadline = clock() + timeout
    last_error = "No HTTP attempt completed"

    def read(path):
        remaining = deadline - clock()
        if remaining <= 0:
            raise SmokeError("Smoke-test deadline exceeded")
        url = urljoin(base_url, path)
        if (urlsplit(url).netloc, urlsplit(url).scheme) != (urlsplit(base_url).netloc, urlsplit(base_url).scheme):
            raise SmokeError("Admin asset is not served by this CMS")
        return get(url, min(10, remaining))

    while clock() < deadline:
        try:
            headers, body = read("health")
            if body.strip() != b"OK":
                raise SmokeError("Invalid liveness response")
            if (commit or revision) and headers.get("Cache-Control") != "no-store":
                raise SmokeError("Liveness response is cacheable")
            if commit and headers.get("X-Armada-Commit") != commit:
                raise SmokeError("Unexpected served commit")
            if revision and headers.get("X-Armada-Revision") != revision:
                raise SmokeError("Unexpected served revision")
            served_revision = headers.get("X-Armada-Revision")
            if accepted_revisions and served_revision and served_revision not in accepted_revisions:
                raise SmokeError("Restored response identifies an unexpected revision")
            for path in ("api/v1/featureflags",
                         "api/v1/featureflags?range=%5B0%2C0%5D&sort=%5B%22key%22%2C%22ASC%22%5D"):
                headers, body = read(path)
                if headers.get("Content-Type", "").split(";")[0] != "application/json":
                    raise SmokeError("API did not return JSON")
                rows = json.loads(body)
                if not isinstance(rows, list) or not all(isinstance(row, dict) for row in rows):
                    raise SmokeError("Invalid feature flag list")
                if not re.fullmatch(r"featureflags \d+-\d+/\d+", headers.get("Content-Range", "")):
                    raise SmokeError("Missing or invalid Content-Range")
                if "?" in path and len(rows) > 1:
                    raise SmokeError("API pagination was ignored")
            headers, body = read("admin/")
            if "text/html" not in headers.get("Content-Type", ""):
                raise SmokeError("Admin HTML is missing")
            html = body.decode("utf-8")
            if not re.search(r'id=["\']root["\']', html):
                raise SmokeError("Admin application root is missing")
            assets = AdminAssets()
            assets.feed(html)
            if not assets.scripts or not assets.styles:
                raise SmokeError("Admin entrypoint assets are missing")
            for paths, types in ((assets.scripts, ("javascript", "ecmascript")),
                                 (assets.styles, ("text/css",))):
                for path in paths:
                    headers, body = read(urljoin("admin/", path))
                    content_type = headers.get("Content-Type", "")
                    if not body or not any(kind in content_type for kind in types):
                        raise SmokeError("Admin asset is missing or returned SPA HTML")
            if clock() >= deadline:
                raise SmokeError("Smoke-test deadline exceeded")
            return
        except Exception as error:
            # Do not print response bodies: even public APIs can contain user data.
            last_error = str(error) if isinstance(error, SmokeError) else type(error).__name__
            remaining = deadline - clock()
            if remaining > 0:
                sleep(min(3, remaining))
    raise SmokeError("CMS smoke test failed: " + last_error)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("url")
    parser.add_argument("--commit")
    parser.add_argument("--revision")
    args = parser.parse_args()
    smoke(args.url, args.commit, args.revision)
    print("CMS smoke test passed")
