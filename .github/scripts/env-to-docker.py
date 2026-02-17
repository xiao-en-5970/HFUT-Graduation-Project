#!/usr/bin/env python3
"""将 .env 转为 Docker --env-file 兼容格式（无行内空格）"""
import re

out = []
with open("/opt/app/.env") as f:
    content = f.read()
for line in content.splitlines():
    line = line.rstrip("\r").strip()
    if not line or line.startswith("#"):
        continue
    m = re.match(r"^([^=]+)=(.*)$", line)
    if m:
        k = m.group(1).strip()
        v = m.group(2).strip().replace("\n", " ").replace("\r", "")
        if " " in v or "\t" in v:
            v = '"' + v.replace("\\", "\\\\").replace('"', '\\"') + '"'
        out.append(f"{k}={v}")
with open("/opt/app/.env.docker", "w") as f:
    f.write("\n".join(out))
