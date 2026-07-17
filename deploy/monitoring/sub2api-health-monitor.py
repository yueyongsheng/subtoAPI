#!/usr/bin/env python3

import json
import os
import shutil
import smtplib
import subprocess
import sys
import time
import urllib.request
from email.message import EmailMessage
from pathlib import Path


STATE_FILE = Path("/var/lib/sub2api-monitor/state.json")
RESTIC_SUCCESS_FILE = Path("/var/lib/sub2api-backup/last-success")
CONTAINERS = ("sub2api", "sub2api-postgres", "sub2api-redis")


def env_bool(name: str, default: bool = False) -> bool:
    value = os.getenv(name)
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "on"}


def check_url(name: str, url: str, issues: list[str]) -> None:
    try:
        request = urllib.request.Request(url, headers={"User-Agent": "sub2api-monitor/1"})
        with urllib.request.urlopen(request, timeout=10) as response:
            if response.status != 200:
                issues.append(f"{name}: HTTP {response.status}")
    except Exception as exc:
        issues.append(f"{name}: {type(exc).__name__}")


def check_containers(issues: list[str]) -> None:
    for container in CONTAINERS:
        try:
            result = subprocess.run(
                [
                    "docker",
                    "inspect",
                    "--format",
                    "{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}",
                    container,
                ],
                check=True,
                capture_output=True,
                text=True,
                timeout=10,
            )
            status, health = result.stdout.strip().split(maxsplit=1)
            if status != "running" or health not in {"healthy", "none"}:
                issues.append(f"container {container}: status={status}, health={health}")
        except Exception as exc:
            issues.append(f"container {container}: inspect failed ({type(exc).__name__})")


def check_disk(issues: list[str]) -> None:
    usage = shutil.disk_usage("/")
    percent = usage.used * 100 / usage.total
    warning = float(os.getenv("DISK_WARNING_PERCENT", "70"))
    critical = float(os.getenv("DISK_CRITICAL_PERCENT", "85"))
    if percent >= critical:
        issues.append(f"disk /: CRITICAL {percent:.1f}% used")
    elif percent >= warning:
        issues.append(f"disk /: WARNING {percent:.1f}% used")


def check_restic_backup(issues: list[str]) -> None:
    maximum_age = int(os.getenv("RESTIC_MAX_AGE_SECONDS", "129600"))
    try:
        completed_at = int(RESTIC_SUCCESS_FILE.read_text(encoding="utf-8").strip())
    except (FileNotFoundError, ValueError):
        issues.append("restic backup: no successful run recorded")
        return
    age = int(time.time()) - completed_at
    if age > maximum_age:
        issues.append(f"restic backup: last success is {age // 3600} hours old")


def load_state() -> dict:
    try:
        return json.loads(STATE_FILE.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError):
        return {}


def save_state(issues: list[str], now: int, sent: bool) -> None:
    previous = load_state()
    STATE_FILE.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "issues": issues,
        "failed": bool(issues),
        "checked_at": now,
        "last_sent_at": now if sent else previous.get("last_sent_at", 0),
    }
    STATE_FILE.write_text(json.dumps(payload, ensure_ascii=True), encoding="utf-8")
    os.chmod(STATE_FILE, 0o600)


def send_email(subject: str, body: str) -> None:
    host = os.environ["SMTP_HOST"]
    port = int(os.getenv("SMTP_PORT", "465"))
    username = os.environ["SMTP_USERNAME"]
    password = os.environ["SMTP_PASSWORD"]
    sender = os.getenv("SMTP_FROM_EMAIL", username)
    recipient = os.environ["ALERT_EMAIL"]

    message = EmailMessage()
    message["Subject"] = subject
    message["From"] = sender
    message["To"] = recipient
    message.set_content(body)

    if env_bool("SMTP_USE_TLS", True) and port == 465:
        with smtplib.SMTP_SSL(host, port, timeout=20) as client:
            client.login(username, password)
            client.send_message(message)
        return

    with smtplib.SMTP(host, port, timeout=20) as client:
        if env_bool("SMTP_USE_TLS", True):
            client.starttls()
        client.login(username, password)
        client.send_message(message)


def main() -> int:
    issues: list[str] = []
    check_containers(issues)
    check_disk(issues)
    check_restic_backup(issues)
    check_url("local health", os.getenv("LOCAL_HEALTH_URL", "http://127.0.0.1:8080/health"), issues)
    check_url("public health", os.getenv("PUBLIC_HEALTH_URL", "https://api-yue88.xyz/health"), issues)

    now = int(time.time())
    previous = load_state()
    previous_issues = previous.get("issues", [])
    repeat_seconds = int(os.getenv("ALERT_REPEAT_SECONDS", "21600"))
    last_sent = int(previous.get("last_sent_at", 0))
    state_changed = issues != previous_issues
    repeat_due = bool(issues) and now - last_sent >= repeat_seconds
    should_send = state_changed or repeat_due
    sent = False

    if should_send:
        if issues:
            subject = "[Yuexiang API] Monitor ALERT"
            body = "Sub2API health checks failed:\n\n" + "\n".join(f"- {issue}" for issue in issues)
        else:
            subject = "[Yuexiang API] Monitor RECOVERED"
            body = "All Sub2API server health checks have recovered."
        try:
            send_email(subject, body)
            sent = True
        except Exception as exc:
            print(f"monitor email failed: {type(exc).__name__}", file=sys.stderr)

    save_state(issues, now, sent)
    if issues:
        print("; ".join(issues), file=sys.stderr)
        return 1
    print("all health checks passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
