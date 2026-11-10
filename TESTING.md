# PISKIN MUSIC — verification and evidence

See [PLAN.md](PLAN.md) for phase status, [ARCHITECTURE.md](ARCHITECTURE.md)
for inspected boundaries, and [DESIGN.md](DESIGN.md) for proposed behavior.

## Recorded baseline — 2026-11-10

Source: `8df198de9e3eccbb717d4370303cdb7f811a72a8`.
Evidence below is from the initial restricted session; unknown and blocked checks
are not passes. No audio recording, credentials or browser profile is committed.

| Check / command | Result |
| --- | --- |
| Git blob hashes against upstream recursive tree | PASS: 78 of 79 upstream files match; only `.gitignore` differs, solely by removing `DESIGN.md`. No missing files; exactly five added root documents. See the provenance recheck below. |
| `git status --short --branch`, `git log`, `git remote -v` | BLOCKED: not a Git repository; `.git` is exposed empty/read-only. |
| `uname -sm` | `Linux aarch64`. |
| `chromium --version` | `Chromium 153.0.8010.36 Arch Linux ARM`. |
| `go version`, `go test ./...` | BLOCKED: `go: command not found`; no tests executed. |
| `pactl info`, `pactl list short sinks`, `pactl list short sources` | BLOCKED: operation not permitted / connection refused. |
| `wpctl status` | BLOCKED: session/system bus operations denied; could not connect to PipeWire. |
| Bounded `pw-record` probe (prior baseline record; not repeated in this review) | BLOCKED: `pw_context_connect() failed: Operation not permitted`; no PCM validated. |
| Real Apple Music subscription playback | NOT VERIFIED: no running build or authenticated playback exercised. |
| Live capture through Go `visualizer.Source` | NOT VERIFIED. Native tool probe cannot substitute for this check. |
| MPRIS runtime / macOS runtime | NOT VERIFIED. |
| Documentation checks | PASS: all five files present; local Markdown targets, trailing whitespace and code-fence balance checked. |
| Phase 0 change review | PASS: hash comparison confirms 78 original files unchanged; only original-file edit removes the DESIGN.md ignore rule. Its no-index diff and whitespace check were inspected. |

The audio denials establish a session access limitation, not a host audio
configuration defect. Do not change the user's PipeWire or Hyprland config to
address sandbox denials.

### Provenance recheck

On 2026-11-10 the review independently fetched the
[commit object](https://api.github.com/repos/k1y0miiii/applemusic-tui/git/commits/8df198de9e3eccbb717d4370303cdb7f811a72a8),
[recursive tree](https://api.github.com/repos/k1y0miiii/applemusic-tui/git/trees/8df198de9e3eccbb717d4370303cdb7f811a72a8?recursive=1),
and [upstream ignore file](https://github.com/k1y0miiii/applemusic-tui/blob/8df198de9e3eccbb717d4370303cdb7f811a72a8/.gitignore)
through the GitHub connector. The commit object confirms tree
`a3a494548ccef67a7b13801b3bf6284a6f8e4a34`; the recursive response is not
truncated and contains 79 blobs. Content comparison found:

- 78 matching blobs, including application source, all tests, dependencies,
  LICENSE, installers, Makefile, workflows, README files and media.
- One changed blob: `.gitignore`, from `665e0352232ddff7a5576c6ce54c9ec9b65f4712`
  to `9ebca5a10a16d98f0c362675198864a0e382f369`.
- Five additional files: `CLAUDE.md`, `ARCHITECTURE.md`, `DESIGN.md`, `PLAN.md`,
  and `TESTING.md`. These documents and the ignore-rule removal existed at
  review entry; this review updates documentation only.
- No missing upstream files or unexpected additions, excluding only the four
  environment-owned directories named in PLAN.md.

The sorted upstream manifest uses UTF-8 lines of `path<TAB>blob_sha<LF>`.
Its SHA-256 is
`a52ba50603f7109060185b1e945a8bb7b0e22a655297027e2768326f24e8a138`.
This digest identifies the comparison input; it is not a signature or proof of
local Git ancestry. File contents were compared; timestamps and permission bits
are not covered by the blob check.

To repeat without Git metadata, save the linked recursive API response as a
JSON file outside the checkout, then run this from the repository root with
Python 3 (replace `/tmp/upstream-tree.json` with that file):

```sh
python3 - /tmp/upstream-tree.json <<'PY'
from pathlib import Path
import hashlib, json, sys

tree = json.loads(Path(sys.argv[1]).read_text())
assert tree.get('truncated') is False
blobs = {x['path']: x['sha'] for x in tree['tree'] if x['type'] == 'blob'}
manifest = ''.join(f'{p}\t{s}\n' for p, s in sorted(blobs.items()))
assert hashlib.sha256(manifest.encode()).hexdigest() == (
    'a52ba50603f7109060185b1e945a8bb7b0e22a655297027e2768326f24e8a138')
for name, expected in sorted(blobs.items()):
    data = Path(name).read_bytes()
    if name == '.gitignore':
        assert b'DESIGN.md' not in data.splitlines()
        data = data.replace(b'PROBLEM.md\n', b'PROBLEM.md\nDESIGN.md\n', 1)
    actual = hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()
    assert actual == expected, name
excluded = {'.git', '.agents', '.codex', '.aws'}
files = {p.as_posix() for p in Path('.').rglob('*')
         if p.is_file() and p.parts[0] not in excluded}
assert files - blobs.keys() == {
    'CLAUDE.md', 'ARCHITECTURE.md', 'DESIGN.md', 'PLAN.md', 'TESTING.md'}
print('PASS: 79 upstream blobs accounted for; only approved phase 0 differences')
PY
```

The no-index diff against the fetched upstream `.gitignore` was reviewed and
contains one deleted line, `DESIGN.md`. Its exit status 1 means a difference
was found; it is expected. Documentation checks cover local Markdown targets,
balanced fences, trailing whitespace and the five required root files. A
separate before/after SHA-256 snapshot checks that this review changed only
the five documents. That review used no-index diffs; repository Git checks
became available after the recovery described below.

### Git recovery — 2026-11-10

After unrestricted access was enabled, the GitHub CLI identified the signed-in
account as `theonheptagrammaton`. The connector and CLI confirmed its owned fork
[piskinmusic](https://github.com/theonheptagrammaton/piskinmusic), push permission,
and the sole branch `main` at the baseline commit. GitHub reports `main` as
unprotected and the applicable branch-rules endpoint returns an empty array.

A clone of this existing repository supplied the original history; no new root
history was initialized. All 79 baseline paths were compared before restoring
`.git`, allowing only the documented ignore-rule removal. `git status` now
shows exactly that modification and the five new root documents. The recovered
`origin` is `https://github.com/theonheptagrammaton/piskinmusic.git`, with local
`main` tracking `origin/main`. The user explicitly authorized commit and push.
The initial Git blocker is resolved; the earlier Go and runtime results remain
unchanged. This recovery did not run playback or capture checks.

## Target verification — 2026-11-10

The earlier restricted-session results above remain historical evidence. This
run used the actual ARM64 desktop with unrestricted session access, starting at
parent commit `9498c01` (upstream application source `8df198d`). The unchanged
source was tested and built before implementation. The implementation and final
results below belong to the commit containing this record.

Environment: Linux `7.2.11-14-1-ARCH`, `aarch64`, Go `1.26.11 linux/arm64`,
Chromium `153.0.8010.36 Arch Linux ARM`, Hyprland `0.56.2`, Ghostty `1.4.0`,
PipeWire/pipewire-pulse `1.6.9`. Go was initially absent; the official ARM64
archive was SHA-256 checked and extracted outside the checkout under
`/tmp/piskin-toolchain/go`. In commands below, `go` denotes `/tmp/piskin-toolchain/go/bin/go`.
No desktop configuration, dependency versions or CDMs changed; sign-in stayed
in the official browser UI.

### Automated baseline and final results

Counts include subtests; top-level counts are shown separately. All six Go
packages ran. Native macOS/Windows integration tests were not executed.

| Command | Unchanged baseline | Final source |
| --- | --- | --- |
| `go mod verify` | PASS, all modules verified | PASS |
| `go vet ./...` | PASS | PASS |
| `go test -json ./...` | 330 PASS, 1 FAIL (237/1 top-level) | 347 PASS, 0 FAIL, 3 SKIP (242/0/3 top-level) |
| `go test -race -json ./...` | Same existing failure; no race reported | 347 PASS, 0 FAIL, 3 SKIP; no race reported |
| `CGO_ENABLED=1 go build ./...` | PASS | PASS |
| `CGO_ENABLED=0 go build -o /tmp/piskin-music-baseline .` | PASS | Final native ARM64 nocgo build PASS |
| `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/piskin-music-linux-arm64 .` | Native Linux ARM64 host | PASS |
| `git diff --check` | Clean at entry | PASS |

The baseline failure was
`TestReadyStartsVisualizerOpenAlongsideInitialStateFetch`: a working desktop bus
added an MPRIS listener (four commands), while the test asserted three capture/
startup commands. Setting an unavailable session-bus address for the unchanged
suite yielded 331 PASS, 0 FAIL for both ordinary and race runs. The final test
isolates its bus environment and retains the exact three-command assertion.
The three final skips are explicit opt-in live browser, live capture and isolated
server tests; their separate executed results follow.

### Gate A — subscription playback on target: PASS

The unchanged baseline TUI opened official headed sign-in, detected login and
parked Chromium on `special:amtui`. The user entered credentials/2FA in the
browser only. Further transport observations used a temporary Go probe outside
the checkout calling the application's existing public `engine.Engine` methods
(`Connect`, `Search`, `Play`, transport/queue/EQ, `Reload`, `Close`). This used the
same headed browser/profile/MusicKit path, without an alternate player. Automatic
TUI retry decisions remain covered by the deterministic reconnect tests.

Initial playback was **not** a pass: the first signed-in application profile
reported `isAuthorized=true` but `hasMusicSubscription()=false`. It played a
90-second preview and later returned `[mk-007] SUBSCRIPTION_ERROR`. The user
confirmed full playback in their usual browser, then signed into the subscribed
account in the application's separate visible profile. Both MusicKit entitlement
checks subsequently returned true. No token injection, cookie export, DRM
workaround or application playback patch was used to change entitlement.

- Full track: Daft Punk / Pharrell Williams / Nile Rodgers, “Get Lucky”, catalog
  ID `617154366`, catalog duration 6:09.629, player duration 6:10. At 21:16:40 TRT
  position was 0:08; at 21:18:44 it was 2:12, continuously playing, not initializing,
  with no player error. The user explicitly confirmed audible output. Read-only
  media diagnostics showed media keys attached, readyState 4 and no media error.
- Chromium remained on `special:amtui` throughout the sustained observation.
- Pause at 21:18:53 stopped playback at 2:21 and yielded zero PCM/FFT levels;
  resume at 21:19:19 restored changing, nonzero PCM/FFT output.
- Append and play-next increased queue length 11 → 12 → 13. Next selected inserted
  ID `1673536443`; Previous returned to `617154366`. Subsequent state and capture
  agreed. Seek to 1:00 was observed at 1:02; volume 40 then 50, shuffle on/off and
  repeat 1/2/0 were confirmed by subsequent snapshots.
- EQ with a nonzero gain attached during protected playback and retained nonzero
  system PCM. Gains were reset afterward. This verifies continued audio with EQ,
  not a calibrated EQ frequency-response measurement.
- Reload retained subscription authorization. Queue replacement started
  `636968288` (“Get Lucky” radio edit), full duration 4:08; Go capture passed.
  Close/Connect with the persistent profile also restored authorized full
  playback and nonzero Go capture without another login.
- The separate disposable headed-browser opt-in test passed under the race
  detector: actual special-workspace placement, a bounded login timeout, then
  killing only its owned browser stopped a pending login in about 6 ms.
  The production five-minute timeout was not waited out. The user's subscribed
  session was not expired or signed out for failure injection.

One Queen catalog item (`6781027645`) returned
`[mk-160] CONTENT_UNAVAILABLE; Unable to load assets for MediaItem` after the
account correction. It is not counted as a playback pass, and its cause was not
established. Other full catalog tracks above passed; no catalog substitution or
DRM workaround was added. This item-level availability observation does not
prevent the verified subscription playback path, but is retained for follow-up.

### Gate B — real Go capture on target: PASS

`pactl info` identified **PulseAudio (on PipeWire 1.6.9)**; `wpctl status`
independently identified PipeWire 1.6.9. The default sink was
`audio_effect.j293-convolver`, with monitor
`audio_effect.j293-convolver.monitor`. The Go source reads the server's actual
`GetServerInfo.PackageName`: `PIPEWIRE` is now used only for an identified
PipeWire server, and `PULSEAUDIO` is the protocol label otherwise. Plain
PulseAudio naming cases are synthetic unit coverage, not a claimed live daemon test.

Final real playback/capture repeat after the startup race fix (TRT, UTC+03:00):
all rows negotiated float32, 48,000 Hz, two channels; all samples were finite.
Times are local read/frame publication timestamps, not hardware sample-clock timestamps.

| State / path | Read interval | Reads / samples | RMS / absolute peak | Service frames / changing spectra |
| --- | --- | --- | --- | --- |
| Playing / Source | 21:31:03.119–21:31:06.085 | 100 / 288,000 | 0.115748 / 0.496281 | — |
| Playing / Service | 21:31:06.172–21:31:09.135 | 100 / 288,000 | 0.114480 / 0.498327 | 99 / 89 |
| Paused / Source | 21:31:26.693–21:31:29.638 | 99 / 285,120 | 0 / 0 | — |
| Paused / Service | 21:31:29.744–21:31:32.710 | 100 / 288,000 | 0 / 0 | 92 / 0 |

The playing and paused runs both passed under `-race`. A final resume repeat
after 21:33:28 also passed with nonzero signal in both paths. Service frame timestamps
advanced and stayed within 400 ms of observation; playing peak band was 0.95508,
paused peak band was zero. Earlier passes also covered resume, track change,
EQ and reload/reconnect. The final source additionally verified canceled reads,
closed reads and reopening through the service subtest. An explicit nonexistent
`PULSE_SERVER` produced the expected immediate unavailable failure. An early
strict signal check started before playback and correctly failed on zero signal;
neither failure was hidden or counted as a successful playback measurement.

A separate 440 Hz, amplitude 0.02 generated tone routed through the real sink
returned RMS approximately 0.014142 and peak 0.02. This was a **test signal**,
not subscription evidence. It was streamed directly, without saving audio.

### Isolated and synthetic failure checks — separate from live gates

`TestPulseIsolatedLifecycle` starts its own PipeWire server and WirePlumber
policy instance in temporary runtime/config/state directories, with two null
sinks and no hardware discovery. It never stops the user's server. Three
consecutive race-enabled runs passed after the fix:

- Reopening after changing the fixture's default sink selected its new monitor.
- Removing that monitor moved capture to the surviving monitor through policy;
  fresh frames continued. Reopening on the surviving sink also succeeded.
- Killing only the fixture server surfaced `PulseAudio server connection: EOF`
  and allowed bounded cleanup. Capture reopened on a new isolated server.

The first race-enabled fixture run exposed competing record-state writes inside
`pulse.RecordStream.Start` and its disconnect callback. The source now sends the
same flush/uncork requests through the public protocol API, checks each error,
and owns lifetime through client close; it does not use those racy state helpers.
Synthetic request regressions cover order, stream identity, uncork and rejection.
No dependency fork or version change was needed.

Synthetic delayed/absent-read checks also failed before the UI boundary fix:
source open claimed LIVE before PCM and stale frames retained LIVE. Starting now
waits for a fresh frame; stale capture is quiet and labeled STALE; fresh zero PCM
remains LIVE; a later fresh frame restores LIVE. Starting, stale and simulated/
unavailable data cannot add invented audio history. Existing terminal-error,
EOF, finite-sample, cancellation and concurrent cleanup regressions still pass.
Canceled login polls previously waited through retries; synthetic before/during
cancellation regressions now pass, alongside the actual owned-browser exit test.

No unresolved general subscription playback or Go capture blocker remained on
the tested target path. Physical hardware hotplug, native macOS/Windows runtime
and deliberate expiration of the user's account session were not performed.
Default-sink/removal/loss cases above are explicitly isolated protocol evidence;
existing captures are selected at open time, and reopening selects the current
monitor. No automatic default-device-following policy is claimed.

### Repeating the opt-in checks

```sh
# For this run’s temporary toolchain (or use your installed Go 1.26+):
export PATH=/tmp/piskin-toolchain/go/bin:$PATH
# Real desktop capture; zero-valued silence is valid.
AMTUI_TEST_PULSE=1 go test -race -v -count=1 -timeout=25s ./visualizer -run '^TestPulseLiveSmoke$'
# During controlled audible playback; requires finite PCM and changing signal.
AMTUI_TEST_PULSE=1 AMTUI_TEST_PULSE_REQUIRE_SIGNAL=1 go test -race -v -count=1 -timeout=25s ./visualizer -run '^TestPulseLiveSmoke$'
# Isolated null-sink protocol fixture, separate from real capture evidence.
AMTUI_TEST_PULSE_FIXTURE=1 go test -race -v -count=3 -timeout=35s ./visualizer -run '^TestPulseIsolatedLifecycle$'
# Disposable headed login/window/cancellation check, no account sign-out.
AMTUI_TEST_LINUX_BROWSER=1 go test -race -v -count=1 -timeout=60s ./engine -run '^TestLinuxHeadedLoginCancellationLive$'
```

Opt-in errors fail rather than skip; flags are off in ordinary suites. Logs
contain aggregate counts, levels, format and timestamps only. No captured PCM,
credentials, browser profiles or screenshots are retained in the repository.

## Initial automated coverage inventory

Static inventory: 33 `_test.go` files declare 260 `Test` functions across the
root, engine, visualizer, lyrics, lastfm and mpris packages. This includes
platform-specific and integration-tagged declarations; it is not an executed
test count. `go version` and `go test ./...` were both attempted again and
returned `go: command not found` (shell exit 127). No suite, vet, race check,
module verification or build ran.

Root tests exercise account, artwork, themes/config, installer behavior, help,
layout, mouse, loading/reconnect, recent items, scrobbling, EQ and visualizer UI.
Package suites cover MusicKit JS/playback lifecycle/window controls, audio
analysis/source/service concurrency, lyrics, Last.fm and MPRIS data behavior.

`engine/musickit_test.go` launches real headless Chromium on `about:blank`, then
tests JS with controlled page fixtures. These are browser-driven tests, not
Apple Music subscription or DRM playback tests. They have no missing-browser
skip guard. Darwin/Windows window integration tests use the `integration` tag;
native CoreAudio smoke testing is opt-in with `AMTUI_TEST_COREAUDIO=1`.
`engine/eq_integration_test.go` is also Darwin/integration-tagged and connects
to real MusicKit playback using the persistent profile. It is separate from
the headless fixtures and needs an authorized desktop/account session.

Existing CI: macOS release workflow runs vet/test then builds release targets;
Windows workflow runs vet/test plus platform integration/install checks. There
is no Linux ARM64 runtime verification job in the inspected workflows.

## Baseline commands when Go is available

Use Go 1.26+ and a Chromium available through the test helper's normal discovery.
`AMTUI_CHROME` controls production discovery, but the headless fixture helper
does not explicitly consult that override. Ensure the test browser is discoverable.

```sh
go version
go mod verify
go vet ./...
go test ./...
go test -race ./...
CGO_ENABLED=0 go build -o /tmp/piskin-music .
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/piskin-music-linux-arm64 .
```

Native Linux tests prove runtime behavior only on the executing architecture.
Cross-compilation proves compilation only. Run `make vet test test-race build
build-nocgo` on a suitable native host. `make verify` also invokes `verify-minos`,
which uses macOS `vtool`; it is not a portable Linux verification command.
On macOS additionally run `make verify`, selected window integration tests and
the opt-in CoreAudio smoke check. A darwin nocgo build is a degraded capture
configuration, not proof of macOS live capture. Preserve Windows checks as well.

## Gate A — real Linux ARM64 Apple Music playback

Run from the actual development checkout with desktop/audio access. Record OS,
architecture, browser version, Go version, compositor/terminal version and
upstream/application commit. Keep account identifiers out of evidence.

1. Build the unchanged baseline. Open the installed headed Chromium normally
   and confirm a subscription catalog track plays through music.apple.com.
   A preview, unencrypted test tone or advancing UI timer is insufficient.
2. Launch the baseline TUI. If login is required, the user signs in only through
   the official browser window, including 2FA. Never request a terminal password.
   Do not automate credential entry or export cookies.
3. Start a full catalog track, confirm audible output and sustained playback
   past the preview boundary for at least two minutes. Record only track/catalog
   identity, duration of observation and redacted media errors if any.
4. Pause/resume, seek, next/previous, queue replacement, append/play-next, volume,
   shuffle/repeat. Confirm sound and state agree. Test EQ without assuming that
   an attached graph proves audible protected playback.
5. Verify the browser remains playable when parked in Hyprland's special
   workspace. Test reload/reconnect and relaunch with session persistence.
6. Test browser cancellation/login timeout without hanging the TUI. Test an
   expired session only with a disposable test session or user authorization
   for sign-out; preserve the user's existing session by default.

Exit requires actual audio and consistent controls. On failure record browser
media/CDM errors without tokens. Do not disable DRM, transfer third-party CDMs
or report a macOS success as a Linux pass.

## Gate B — real PipeWire capture

1. In the accessible user session, inspect `pactl info`, sinks and monitor sources
   and `wpctl status`. Confirm whether the server is PipeWire's PulseAudio
   compatibility service or plain PulseAudio. The existing `PIPEWIRE` source
   name is hardcoded and does not prove server identity.
2. Use an opt-in Go smoke test to open `visualizer.OpenSystemSource()` and
   `NewService()`, with a bounded timeout and guaranteed cleanup. Record source,
   actual sample rate/channels, nonempty finite reads, fresh frame times and
   nonzero signal during playback. Do not save raw PCM.
3. During audible Apple Music playback, observe changing PCM/FFT output. Pause
   it, silence other applications and verify levels settle. A monitor can
   capture other applications, so correlate the source with controlled playback.
4. A known tone played through the real sink may independently verify routing
   and frequency response. Label it a test signal; it does not prove DRM playback.
   Fresh zero-valued PCM is valid silence, not capture failure.
5. Test output-device changes, disappearing server/monitor, delayed reads,
   cancellation, EOF and re-opening. Do not stop the user's audio server for a
   test; use isolated fixtures for destructive failure injection.
6. Verify the UI cannot retain LIVE after terminal failure or stale capture.
   Starting/unavailable/demo must never feed invented audio history.

Pass only after the application Go backend captures actual output PCM. The
native `pw-record` probe and the Go backend are separate checks. Gate B and
Gate A must both pass before major UI development.

## Per-phase regression checks

| Phase | Required evidence |
| --- | --- |
| Documentation | Five documents present, local links valid, original source unchanged, ignore rule reviewed. |
| Playback/capture fixes | Relevant engine/source/service regressions, baseline suite and both live gates. |
| State separation | Existing suites, injectable playback errors, no duplicate reconnect/poll/capture workers, race check. |
| Screens/theme | 40x12, 60x20, 80x24, 120x32, 180x50 and smaller-than-minimum; keyboard/mouse parity, Unicode, overlay precedence and resize; monochrome except art. |
| PCM waveform | Silence, sine, impulse, anti-phase stereo, irregular chunks, finite bounds, frame ownership, stale/error/demo behavior; real capture repeat. |
| Optimization/release | Profile before/after, native platform tests, build matrix, packaging/license checks and migration review. |

For each implemented phase record commands, commit, actual pass/fail/skip counts,
runtime evidence, diff review and remaining failures. Do not weaken existing
assertions merely to make a refactor pass.

## Resource and feature acceptance

Measure Go process and Chromium separately over the same track and layout:
startup/first-audio time, input latency, CPU, RSS, redraw rate and capture
freshness. Compare playing/paused and each screen. Target responsive input
(p95 under 100 ms) and no more than 30 visual redraws/s; these are proposed
targets, not measurements. Investigate idle redraws, retained frame growth and
audio discontinuities. Set CPU/RSS budgets only after measuring the baseline.

Manually verify search/library/recent, all queue operations, lyrics, artwork,
EQ persistence, account controls, saved settings, Last.fm opt-in, media keys and
`playerctl -p amtui` controls. No feature is removed without documented intent.

## Commit and push checklist

Inspect `git diff --check`, changed-file list and full diff after each phase.
Never stage profiles, credentials, audio captures or unrelated user edits.
Commit completed phases with descriptive messages. Inspect branch, remotes and
tracking destination before pushing; use only the configured development branch.
The initial missing-metadata blocker was resolved by recovering the fork's
existing history. Keep the restricted-session results above as historical
evidence, and verify the remote commit and a clean working tree after pushing.
