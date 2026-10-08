#!/usr/bin/env python3
"""Presses Ctrl-C on a command once a log file contains a pattern (E2E).

    python3 scripts/e2e_interrupt.py LOGFILE PATTERN -- fpack build dmg

Runs the command in its own process group (like a terminal's foreground
job), waits until LOGFILE contains PATTERN, sends SIGINT to the whole group
(what Ctrl-C does) and exits with the command's exit code. Output is written
to stdout/stderr unchanged.
"""
import os
import signal
import subprocess
import sys
import time


def main() -> int:
    log, pattern = sys.argv[1], sys.argv[2]
    cmd = sys.argv[sys.argv.index("--") + 1:]
    p = subprocess.Popen(cmd, start_new_session=True)
    deadline = time.time() + 1200
    while time.time() < deadline and p.poll() is None:
        if os.path.exists(log) and pattern in open(log, encoding="utf-8", errors="replace").read():
            time.sleep(3)
            print(f"[e2e] pressing Ctrl-C (pattern {pattern!r} seen)", file=sys.stderr, flush=True)
            os.killpg(p.pid, signal.SIGINT)
            break
        time.sleep(1)
    else:
        if p.poll() is None:
            p.kill()
        print(f"[e2e] pattern {pattern!r} never appeared in {log}", file=sys.stderr)
        p.wait()
        return 99
    return p.wait()


if __name__ == "__main__":
    sys.exit(main())
