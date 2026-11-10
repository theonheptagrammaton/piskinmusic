#!/usr/bin/env python3
"""Compare one immutable harness on baseline/current source on this Linux target.

GO=/path/to/go python3 scripts/measure-performance.py BASE_REF /tmp/evidence
Run without other build/benchmark jobs. No browser or audio session is opened.
"""
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parent.parent
baseline = sys.argv[1] if len(sys.argv) > 1 else "1e2b3be"
output = Path(sys.argv[2] if len(sys.argv) > 2 else "/tmp/piskin-performance").resolve()
output.mkdir(parents=True, exist_ok=True)
env = dict(os.environ, GOMAXPROCS="4", AMTUI_MEASURE_UI="1")
go = env.get("GO", "go")
subprocess.run(["git", "rev-parse", "--verify", baseline + "^{commit}"], cwd=root, check=True)
with tempfile.TemporaryDirectory(prefix="piskin-measure-") as temp:
    temporary = Path(temp)
    before = temporary / "baseline"
    subprocess.run(["git", "worktree", "add", "--detach", str(before), baseline], cwd=root, check=True)
    try:
        shutil.copy2(root / "performance_test.go", before / "performance_test.go")
        for label, source in (("before", before), ("after", root)):
            binary = temporary / (label + ".test")
            subprocess.run([go, "test", "-c", "-o", str(binary), "."], cwd=source, env=env, check=True)
            for workload, args in (
                ("bench", ["-test.run=^$", "-test.bench=^BenchmarkTarget", "-test.benchmem", "-test.benchtime=1s", "-test.count=5"]),
                ("cadence", ["-test.run=^TestTargetCadenceMeasurement$", "-test.v"]),
            ):
                with (output / f"{label}-{workload}.txt").open("w") as log:
                    process = subprocess.Popen([str(binary), *args], cwd=source, env=env,
                                               stdout=log, stderr=subprocess.STDOUT)
                    _, status, usage = os.wait4(process.pid, 0)
                    process.returncode = os.waitstatus_to_exitcode(status)
                    log.write(f"\nProcess user_s={usage.ru_utime:.6f} system_s={usage.ru_stime:.6f} max_rss_KiB={usage.ru_maxrss}\n")
                    if process.returncode:
                        raise subprocess.CalledProcessError(process.returncode, process.args)
                print(f"Measured {label} {workload}", flush=True)
    finally:
        subprocess.run(["git", "worktree", "remove", "--force", str(before)], cwd=root, check=True)
