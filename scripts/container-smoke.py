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
env = dict(os.environ)
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
        reported_version = subprocess.check_output(
            ["docker", "exec", container, "/app/pikpak-rss-manager", "version"], text=True).strip()
        expected_version = os.environ.get("SMOKE_EXPECTED_VERSION")
        if expected_version:
            assert reported_version == expected_version, reported_version
        assert request("/healthz")["version"] == reported_version
        assert request("/api/session")["version"] == reported_version
        version_label = '<span class="version">' + reported_version + '</span>'
        with opener.open(base + "/", timeout=10) as page:
            assert version_label in page.read().decode("utf-8"), "login version missing"
        try:
            request("/api/subscriptions")
            raise AssertionError("anonymous management API access allowed")
        except urllib.error.HTTPError as error:
            assert error.code == 401
        assert request("/api/session")["initialized"] is False
        csrf = request("/api/session")["csrf"]
        request("/api/setup", "POST", {"password": password, "confirm_password": password,
                                       "public_url": base, "allow_private_feeds": False})
        assert request("/api/session")["initialized"] is True
        csrf = request("/api/session")["csrf"]
        try:
            request("/api/setup", "POST", {"password": "overwrite", "confirm_password": "overwrite"})
            raise AssertionError("completed setup could be overwritten")
        except urllib.error.HTTPError as error:
            assert error.code == 409
        with opener.open(base + "/", timeout=10) as page:
            assert version_label in page.read().decode("utf-8"), "dashboard version missing"
        subscription = request("/api/subscriptions", "POST", {
            "name": "持久化測試", "rss_url": "https://example.org/feed?private=ci-only",
            "destination": "Test", "enabled": False, "interval_minutes": 10,
            "rename_enabled": False, "rename_mode": "replace", "regex": "", "replacement": ""})
        assert subscription["rename_enabled"] is False
        preview = request("/api/rules/preview", "POST", {
            "rule": {"title": "命名測試", "rename_enabled": True, "mode": "replace",
                     "regex": r"^(?P<title>.+)_(?P<ep>\d+)\.(?P<ext>[^.]+)$",
                     "replacement": "${title} - E${ep}.${ext}"},
            "filename": "作品_03.mkv"})
        assert preview["old_name"] == "作品_03.mkv"
        assert preview["name"] == "作品 - E03.mkv" and preview["matched"] is True
        assert preview["raw_name"] == preview["name"]
        adjusted = request("/api/rules/preview", "POST", {
            "rule": {"title": "命名測試", "rename_enabled": True, "mode": "replace",
                     "regex": r"\[(\d+)\]", "replacement": "S01E$1"},
            "filename": "中文 / English [01].mkv"})
        assert adjusted["raw_name"] == "中文 / English S01E01.mkv"
        assert adjusted["name"] == "中文 _ English S01E01.mkv" and adjusted["warnings"]
        compose("down")  # volume and AES key are intentionally retained
        compose("up", "-d", "--pull", "never")
        ready()
        assert request("/api/session")["initialized"] is True
        login()
        restored = request("/api/subscriptions")
        assert len(restored) == 1 and restored[0]["id"] == subscription["id"]
        assert restored[0]["rss_url"].endswith("private=ci-only")
        assert restored[0]["rename_enabled"] is False and restored[0]["rename_mode"] == "replace"
        print("PASS: Compose startup without env file, version " + reported_version + ", Web setup, hashed-password login after restart, Regex preview, non-root and encrypted data persistence")
    finally:
        compose("down", "--volumes", "--remove-orphans")
