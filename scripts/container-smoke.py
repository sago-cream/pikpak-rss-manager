#!/usr/bin/env python3
"""Run the shipped Compose configuration with an image override and no PAT.

The test exercises login, CSRF, SQLite/key persistence, health and non-root
execution. No PikPak account or external RSS source is contacted.
"""
import http.cookiejar
import json
import os
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
import uuid

image = sys.argv[1] if len(sys.argv) > 1 else "ghcr.io/wade00754/pikpak-rss-manager:latest"
project = "rss-smoke-" + uuid.uuid4().hex[:8]
# A short random password exercises the absence of a minimum-length rule.
password = uuid.uuid4().hex[:8]
env = dict(os.environ, APP_ADMIN_PASSWORD=password, APP_PUBLIC_URL="")
jar = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
base = "http://127.0.0.1:8080"
csrf = ""


def request(path, method="GET", value=None):
    body = None if value is None else json.dumps(value).encode()
    req = urllib.request.Request(base + path, body, method=method, headers={
        "Content-Type": "application/json", "X-CSRF-Token": csrf, "Origin": base})
    with opener.open(req, timeout=10) as result:
        return json.load(result)


def ready():
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        try:
            assert request("/healthz")["status"] == "ok"
            return
        except (OSError, AssertionError):
            time.sleep(1)
    raise RuntimeError("container health timeout")


def login():
    global csrf
    csrf = request("/api/session")["csrf"]
    request("/api/login", "POST", {"password": password})


with tempfile.TemporaryDirectory() as directory:
    override = os.path.join(directory, "override.yml")
    with open(override, "w", encoding="utf-8") as stream:
        # JSON is valid YAML, and safely quotes even digest-based image names.
        service = {"image": image}
        if os.environ.get("SMOKE_PLATFORM"):
            service["platform"] = os.environ["SMOKE_PLATFORM"]
        json.dump({"services": {"pikpak-rss-manager": service}}, stream)
    command = ["docker", "compose", "-p", project, "-f", "docker-compose.yml", "-f", override]

    def compose(*args):
        return subprocess.check_output(command + list(args), env=env, text=True).strip()

    try:
        compose("config", "--quiet")
        compose("up", "-d", "--pull", "never")
        ready()
        container = compose("ps", "-q", "pikpak-rss-manager")
        user = subprocess.check_output(["docker", "inspect", "--format", "{{.Config.User}}", container], text=True).strip()
        assert user == "65532:65532", user
        subprocess.check_call(["docker", "exec", container, "/app/pikpak-rss-manager", "healthcheck"])
        try:
            request("/api/subscriptions")
            raise AssertionError("anonymous management API access allowed")
        except urllib.error.HTTPError as error:
            assert error.code == 401
        login()
        subscription = request("/api/subscriptions", "POST", {
            "name": "持久化測試", "rss_url": "https://example.org/feed?private=ci-only",
            "destination": "Test", "enabled": False, "interval_minutes": 10,
            "season": 1, "regex": "(?P<ep>[0-9]+)", "template": "{title} - E{ep:02}.{ext}"})
        compose("down")  # volume and AES key are intentionally retained
        compose("up", "-d", "--pull", "never")
        ready()
        login()
        restored = request("/api/subscriptions")
        assert len(restored) == 1 and restored[0]["id"] == subscription["id"]
        assert restored[0]["rss_url"].endswith("private=ci-only")
        print("PASS: shipped Compose startup, health, authentication, non-root and encrypted data persistence")
    finally:
        compose("down", "--volumes", "--remove-orphans")
