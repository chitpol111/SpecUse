"""Verify guide fixture quotes with the real Go guest API on loopback.

The catalog is exported from a disposable SQL test. This does not test native
Supabase Auth, FE, or a hosted project. No credentials are used.
"""
import copy
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request


def main():
    root, binary, catalog = map(Path, sys.argv[1:])
    with socket.socket() as probe:
        probe.bind(("127.0.0.1", 0))
        port = probe.getsockname()[1]
    env = {k: v for k, v in os.environ.items() if not k.startswith("SUPABASE_")}
    env.update(APP_ENV="local", CATALOG_BACKEND="memory",
               LOCAL_CATALOG_FILE=str(catalog), PORT=str(port))
    process = subprocess.Popen([str(binary)], cwd=root, env=env,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    base = f"http://127.0.0.1:{port}/api/v1"

    def validate(body):
        payload = {k: body[k] for k in ("items", "ram_items")}
        request = urllib.request.Request(base + "/builds/validate",
                  data=json.dumps(payload).encode(),
                  headers={"Content-Type": "application/json"}, method="POST")
        try:
            response = urllib.request.urlopen(request, timeout=5)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            return response.status, json.load(response)

    try:
        for _ in range(100):
            try:
                with urllib.request.urlopen(base + "/health", timeout=1) as response:
                    if response.status == 200:
                        break
            except (OSError, urllib.error.URLError):
                if process.poll() is not None:
                    raise AssertionError("Fixture API failed to start")
                time.sleep(0.1)
        else:
            raise AssertionError("Fixture API readiness timed out")
        examples = root / "examples" / "drafts"
        for name, expected in [("e2e-baseline", 3600), ("e2e-multiple", 5600),
                               ("e2e-ram-owned", 2500)]:
            body = json.loads((examples / f"{name}.request.json").read_text())
            status, result = validate(body)
            assert status == 200 and result["data"]["compatible"] is True, name
            assert result["data"]["total_price_thb"] == expected, name
        for name in ("e2e-inactive", "e2e-ram-unavailable"):
            body = json.loads((examples / f"{name}.request.json").read_text())
            status, result = validate(body)
            assert status == 404 and result["error"]["code"] == "NOT_FOUND", name
        baseline = json.loads((examples / "e2e-baseline.request.json").read_text())
        mismatch = copy.deepcopy(baseline)
        mismatch["items"][0]["product_id"] = "10101010-0000-4000-8000-000000000002"
        status, result = validate(mismatch)
        assert status == 200 and result["data"]["compatible"] is False
        assert result["data"]["total_price_thb"] == 3800
        assert any(i["code"] == "CPU_SOCKET_MISMATCH" for i in result["data"]["issues"])
        mismatch = copy.deepcopy(baseline)
        mismatch["ram_items"][0]["ram_model_id"] = "20202020-0000-4000-8000-000000000004"
        status, result = validate(mismatch)
        assert status == 200 and result["data"]["compatible"] is False
        assert result["data"]["total_price_thb"] == 3850
        assert any(i["code"] == "RAM_TYPE_MISMATCH" for i in result["data"]["issues"])
        print("PASS: fixture Go API quotes 3600/5600/2500, inactive/RAM unavailable 404, CPU/RAM conflicts")
    finally:
        if process.poll() is None:
            process.send_signal(signal.SIGTERM)
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()


if __name__ == "__main__":
    main()
