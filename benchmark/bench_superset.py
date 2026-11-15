"""Time the benchmark queries through Superset's SQL Lab API (sync mode)."""

import json
import os
import sys
import time
import urllib.request

base = sys.argv[1].rstrip("/")
pw = os.environ["SUPERSET_ADMIN_PASSWORD"]


def call(path, body=None, token=None):
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(base + path, data=data)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    with urllib.request.urlopen(req, timeout=180) as r:
        return json.load(r)


login = {"username": "admin", "password": pw, "provider": "db", "refresh": True}
tok = call("/api/v1/security/login", login)["access_token"]
dbs = call("/api/v1/database/", token=tok)["result"]
db_id = next(d["id"] for d in dbs if d["database_name"] == "hst_dataset")

for line in open(sys.argv[2], encoding="utf-8"):
    name, sql = line.rstrip("\n").split("\t")
    for run in (1, 2):
        t0 = time.time()
        body = {
            "database_id": db_id,
            "sql": sql,
            "runAsync": False,
            "queryLimit": 1000,
            "client_id": f"b{int(t0 * 1000) % 10**9}",
            "tab": "bench",
        }
        try:
            status = call("/api/v1/sqllab/execute/", body, tok).get("status", "?")
        except Exception as err:  # report and continue with the next query
            status = "error: " + str(err)[:80]
        print(f"{name}\trun{run}\t{(time.time() - t0) * 1000:.0f} ms\t{status}", flush=True)
