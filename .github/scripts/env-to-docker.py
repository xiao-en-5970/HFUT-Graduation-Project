#!/usr/bin/env python3
"""将 .env 转为 Docker --env-file 兼容格式（无行内空格）"""
import re
import sys

out = []
with open("/opt/app/.env") as f:
    content = f.read()
for line in content.splitlines():
    line = line.rstrip("\r").strip()
    if not line or line.startswith("#"):
        continue
    m = re.match(r"^([A-Za-z_][A-Za-z0-9_]*)=(.*)$", line)
    if m:
        k = m.group(1).strip()
        v = m.group(2).strip().replace("\n", " ").replace("\r", "")
        if " " in v or "\t" in v:
            v = '"' + v.replace("\\", "\\\\").replace('"', '\\"') + '"'
        out.append(f"{k}={v}")

if not any(kv.startswith("DB_HOST=") for kv in out):
    sys.stderr.write("error: .env 缺少 DB_HOST，无法连接数据库\n")
    sys.exit(1)

with open("/opt/app/.env.docker", "w") as f:
    f.write("\n".join(out))
