#!/usr/bin/env python3
"""Drive a compiled Go test binary through a Linux PTY, without account access."""
import fcntl
import os
import pty
import select
import signal
import struct
import subprocess
import sys
import termios
import time

master, slave = pty.openpty()


def resize(cols, rows):
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))


resize(120, 40)
process = subprocess.Popen([sys.argv[1], "-test.run=^TestTerminalAcceptance$", "-test.v", "-test.timeout=20s"],
                           stdin=slave, stdout=slave, stderr=slave,
                           env=dict(os.environ, TERM="xterm-256color", COLORTERM="truecolor", AMTUI_TERMINAL_ACCEPTANCE="1"))
output = bytearray()


def drain(seconds=0.25):
    end = time.monotonic() + seconds
    while time.monotonic() < end:
        if select.select([master], [], [], max(0, end-time.monotonic()))[0]:
            output.extend(os.read(master, 65536))


def send(data):
    os.write(master, data)
    drain()


try:
    drain(0.5)
    send(b"2")                  # keyboard Focus
    send(b" ")                  # keyboard pause
    send(b"\x1b[<0;6;39M")      # SGR mouse play (120x40 transport)
    send(b"\x1b[<0;35;39M")     # progress seek
    send(b"3")                  # Visualizer
    send(b"?")                  # blocking help
    send(b"\x1b")               # dismiss help
    send(b"\x1b[<0;38;2M")      # navigation: Lyrics (after active Viz label)
    send(b"1")                  # Library
    send(b"c")                  # queue overlay
    send(b"\x1b")               # close queue
    send(b"\x1b[O")             # terminal blur
    resize(30, 8)
    process.send_signal(signal.SIGWINCH)
    drain()
    resize(120, 40)
    process.send_signal(signal.SIGWINCH)
    send(b"\x1b[I")             # focus refresh
    send(b"q")
    drain(0.5)
    process.wait(timeout=5)
    # Print only the test report, excluding ANSI fixture screen output.
    text = output.decode(errors="replace")
    for line in text.splitlines():
        if "terminal_acceptance_test.go:" in line or "--- PASS:" in line or "--- FAIL:" in line:
            print(line[line.find("terminal_acceptance_test.go:"):] if "terminal_acceptance_test.go:" in line else line)
    if process.returncode:
        sys.exit(process.returncode)
finally:
    if process.poll() is None:
        process.kill()
        process.wait()
    os.close(master)
    os.close(slave)
