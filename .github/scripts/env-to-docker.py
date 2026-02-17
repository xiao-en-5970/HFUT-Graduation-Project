#!/usr/bin/env python3
"""将 .env 转为 Docker --env-file 兼容格式"""
import re
import sys

out = []
with open("/opt/app/.env") as f:
    for line in f:
        line = line.rstrip("\r\n").strip()
        if not line or line.startswith("#"):
            continue
        m = re.match(r"^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$", line)
        if m:
            k, v = m.group(1), m.group(2).strip().replace("\n", " ")
            if " " in v or "\t" in v:
                v = '"' + v.replace("\\", "\\\\").replace('"', '\\"') + '"'
            out.append(f"{k}={v}")

if not any(kv.startswith("DB_HOST=") for kv in out):
    sys.stderr.write("error: .env 缺少 DB_HOST\n")
    sys.exit(1)

with open("/opt/app/.env.docker", "w") as f:
    f.write("\n".join(out))
