#!/usr/bin/env python3
"""Reproduce release notices from the pinned module cache and Go toolchain."""
import json
import os
from pathlib import Path
import subprocess
import sys

root = Path(__file__).resolve().parent.parent
go = os.environ.get("GO", "go")


def run(*args):
    return subprocess.check_output([go, *args], cwd=root, text=True)


requirements = json.loads(run("mod", "edit", "-json"))["Require"]


def notice_text(path):
    # Preserve notice wording; normalize only end-of-line whitespace.
    return "\n".join(line.rstrip() for line in path.read_text().splitlines()).rstrip()


notices = [
    "PISKIN MUSIC — third-party notices\n",
    "Generated from every direct/indirect go.mod requirement, including Windows.\n"
    "The application license and upstream attribution are in LICENSE.\n"
    "Toolchain/runtime notices follow the modules. No browser, CDM or fonts are bundled.\n",
]
for requirement in sorted(requirements, key=lambda m: m["Path"]):
    module = json.loads(run("list", "-m", "-json", requirement["Path"]))
    if "Dir" not in module:
        sys.exit(f"Module unavailable: {module['Path']}; run go mod download first")
    directory = Path(module["Dir"])
    files = sorted(p for p in directory.iterdir() if p.is_file() and
                   p.name.upper().startswith(("LICENSE", "COPYING", "NOTICE", "AUTHORS", "PATENTS")))
    # Gonum carries notices for incorporated algorithms as well as its own BSD license.
    third_party = directory / "THIRD_PARTY_LICENSES"
    if third_party.is_dir():
        files.extend(sorted(p for p in third_party.rglob("*") if p.is_file()))
    if not files:
        # This pinned Windows decoder declares MIT and the author in README,
        # but ships no separate license text. Preserve that declaration exactly.
        if module["Path"] != "github.com/mattn/go-localereader":
            sys.exit(f"No license material for {module['Path']}")
        files = [directory / "README.md"]
    notices.append(f"\n{'=' * 72}\n{module['Path']} {module['Version']}\n")
    for path in files:
        notices.append(f"\n--- {path.relative_to(directory)} ---\n{notice_text(path)}\n")

goroot = Path(run("env", "GOROOT").strip())
notices.append(f"\n{'=' * 72}\nGo toolchain/runtime {run('env', 'GOVERSION').strip()}\n")
for name in ("LICENSE", "PATENTS"):
    path = goroot / name
    if path.is_file():
        notices.append(f"\n--- {name} ---\n{notice_text(path)}\n")
result = "".join(notices)
target = root / "THIRD_PARTY_NOTICES.txt"
if "--check" in sys.argv:
    if not target.exists() or target.read_text() != result:
        sys.exit("THIRD_PARTY_NOTICES.txt is out of date; regenerate with the release toolchain")
else:
    target.write_text(result)
print(f"Verified notices for {len(requirements)} modules and the Go runtime")
