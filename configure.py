#!/usr/bin/env python3
"""Configure MiyooDrome on a computer before ejecting the SD card."""
import argparse
import getpass
import hashlib
import json
import os
from pathlib import Path
import secrets
import tempfile
from urllib.parse import urlsplit


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("sd_root", nargs="?", type=Path, default=Path(__file__).resolve().parent,
                        help="SD card or extracted folder containing App/MiyooDrome")
    root = parser.parse_args().sd_root
    app = root / "App" / "MiyooDrome"
    if not (app / "MiyooDrome").is_file():
        parser.exit(1, "Choose the folder containing the installed App/MiyooDrome.\n")
    target = app / "data" / "config.json"
    if target.exists() and input("Replace existing MiyooDrome connection? [y/N] ").lower() != "y":
        return
    server = input("Navidrome base URL (include http:// or https:// and any port): ").strip()
    url = urlsplit(server)
    if (url.scheme not in ("http", "https") or not url.hostname or url.username is not None
            or url.query or url.fragment):
        parser.exit(1, "Enter the base URL without embedded credentials, a query, or a fragment.\n")
    username = input("Navidrome username: ").strip()
    password = getpass.getpass("Navidrome password: ")
    if not username or not password:
        parser.exit(1, "A username and password are required.\n")
    salt = secrets.token_hex(16)
    token = hashlib.md5((password + salt).encode("utf-8")).hexdigest()
    config = dict(server=server.rstrip("/"), username=username, salt=salt, token=token, max_bitrate=192)
    target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=target.parent, delete=False, encoding="utf-8") as out:
        tmp = Path(out.name)
        try:
            json.dump(config, out, indent=2)
            out.write("\n")
            out.flush()
            os.fsync(out.fileno())
        except BaseException:
            tmp.unlink(missing_ok=True)
            raise
    try:
        os.replace(tmp, target)
    finally:
        tmp.unlink(missing_ok=True)
    print("Saved MiyooDrome connection. On the handheld, open Navidrome → Test connection.")


if __name__ == "__main__":
    main()
