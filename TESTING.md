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

## Phase 2 — root responsibility extraction — 2026-11-10

Parent: `e816b69` on local `main`, tracking the configured development fork
`origin/main`. The working tree was clean at entry. The pre-extraction
`go test ./...` passed across all six packages. This phase changes only root
application files and documentation; all existing tests and the `engine`,
`visualizer`, `mpris`, `lyrics` and `lastfm` packages remain unchanged.

Using `/tmp/piskin-toolchain/go/bin/go` (Go 1.26.11, Linux ARM64):

| Check | Result |
| --- | --- |
| `go test ./...` | PASS, existing regression suite plus new command/failure/capture-status/resize checks |
| `go test -race ./...` | PASS, no race reported |
| `go vet ./...` | PASS |
| `CGO_ENABLED=1 go build ./...` | PASS |
| `CGO_ENABLED=0 go build ./...` | PASS |
| Render comparison against parent `e816b69` | PASS, all 60 SHA-256 frame hashes match |
| `git diff --check` | PASS |

The temporary render comparison ran the same test probe against an archive of
the parent and the extracted source: 12 scenes at 70×22, 80×24, 120×35, 200×50
and 69×35, with a fixed ASCII color profile and animation time. Scenes cover the
queue, recent grid, search/help/account/EQ overlays, each capture label, boot and
failure. The probe and archive stayed outside the committed artifacts.

New tests execute root commands against fake playback and per-model command
factories. They verify deferred calls, captured engine/query identity, action
and read failures, close-before-connect ordering, failed startup, stale metadata
identity, one poll per state response, one state read per poll and capture reuse
on reconnect. Rendering, resize and UI ticks add no playback calls or source
opens. Capture status distinguishes starting, live silence, stale, unavailable
and demo while retaining the existing simulated fallback.

Resize checks shrink below and grow above dashboard/grid thresholds, verify
rendered tile captions against their hit rectangles, retain selection, and move
focus away from a hidden recent grid. Hidden controls, borders, gaps and blank
padding are inert. These are the only intentional input corrections; dashboard
output and visible playback interactions are preserved.

No additional browser polling, playback session, source capture or desktop
configuration work was performed. Prior target playback/capture evidence remains
unchanged; native macOS/Windows runtime checks were not repeated.

## Phase 3 — monochrome shell and four screens — 2026-11-10

Parent: `c0d4b2c`, on local `main` tracking the configured development fork
`origin/main`. The working tree was clean at entry; the baseline `go test ./...`
passed. This phase intentionally replaces the dashboard with Library, Focus,
Visualizer and Lyrics, adds PISKIN MUSIC display branding and makes the new
`piskin` palette the monochrome default. It does not change the module path,
executable/installer aliases, authentication profiles, MPRIS IDs, playback
lifecycle or capture service. No dependency was added; artwork still uses the
original-color half-block renderer.

Using `/tmp/piskin-toolchain/go/bin/go` (Go 1.26.11, native Linux ARM64):

| Check | Result |
| --- | --- |
| `go test ./... -count=1` | PASS, 385 test/subtest passes, 3 opt-in skips, 0 failures; all 6 packages pass |
| `go test -race ./... -count=1` | PASS, 385 test/subtest passes, 3 opt-in skips, 0 failures; all 6 packages pass |
| `go vet ./...` | PASS |
| `CGO_ENABLED=1 go build -o /tmp/piskin-phase3 .` | PASS |
| `CGO_ENABLED=0 go build -o /tmp/piskin-phase3-nocgo .` | PASS |
| `/tmp/piskin-phase3 --version` | PASS, PISKIN MUSIC display name; executable remains amtui |
| `git diff --check` | PASS |

The three skipped opt-in tests are `TestPulseLiveSmoke`,
`TestPulseIsolatedLifecycle`, and `TestLinuxHeadedLoginCancellationLive`.
No subscription playback or actual capture verification was repeated for this
UI phase; the prior target evidence remains separate. Native macOS/Windows,
interactive terminal input, and performance profiling were not repeated.

Deterministic coverage includes:

- All four screens at 40×12, 60×20, 80×24, 120×32, 180×50, wide/short 120×12,
  and smaller/zero dimensions. Search, help, account, EQ and queue are resized
  while open. Unicode/combining/emoji titles, exact frame bounds and nonempty
  hit-rectangle bounds are checked. Offscreen pointer input cannot activate a
  control, including tiny overlay frames.
- Keyboard/click screen parity, forward/reverse visible-region focus, retained
  per-screen focus, queue/recent/library/search selections and search results.
  Digits remain query text in search and cannot change screens through blocking
  overlays. Every screen has queue/search/EQ/account/help access.
- Scrolled library/search rows are checked against actual rendered titles, then
  their pointer commands execute against fake playback. Search append/play-next,
  category clicks, transport pause/seek/shuffle/repeat/volume, EQ controls and
  explicit account sign-out all retain keyboard command paths. General help
  commands and both keys in paired entries are reachable by pointer. Narrow
  help pages preserve every action without truncation.
- True-color shell escape sequences are checked for equal RGB channels in all
  default screens and overlays, boot and failure views. A red/blue image retains
  its foreground/background colors through half-block rendering inside Focus;
  artwork cannot recolor PISKIN shell tokens. ASCII rendering retains textual
  screen/focus/selection/playing markers.
- All six original palette names, saved palette precedence, custom legacy
  overrides and existing color-only configs retain their behavior. Explicit or
  saved `piskin` stays fixed monochrome. Theme resolution neither writes the
  config nor changes its saved palette file. Preference writes occur only on
  explicit user theme/EQ/viz actions.

Existing focus tests now include the visible Library list; the visualizer View
assertion explicitly selects the Visualizer screen. The zero-size resize check
accepts an empty frame. Existing queue/recent playback, initialization,
reconnect, artwork, EQ, account, installer and package regression assertions
remain in place. Text frame review inspected all four screens at 40×12 and
120×32 using a temporary ASCII render probe, removed after review. This was a
render review, not a claim of interactive playback or hardware validation.

The full diff was reviewed for geometry/command consistency and persisted-state
compatibility. Only root UI code/tests and product/config documentation changed.
No browser/profile, audio recording, generated binary or credential is committed.
PCM waveform, explicit demo policy and resource/platform validation remain later
phases. This phase is committed locally; no push is part of this request.


## Phase 4 — PCM waveform and capture truth — 2026-11-10

Parent: `1155b77`, on local `main` tracking the configured development fork
`origin/main`. Entry tree was clean. The unchanged `go test ./...` baseline
passed. Go was absent from PATH; all commands below used the existing official
`/tmp/piskin-toolchain/go/bin/go` (Go 1.26.11, native Linux ARM64).

This phase intentionally replaces automatic simulated fallback with quiet
UNAVAILABLE panels. `[visualizer] demo = true` explicitly restores the legacy
DEMO · SIMULATED bands only when capture is unavailable. Starting/stale capture
remains quiet; demo supplies neither PCM nor recorded track history. Existing
fallback/title/history/error tests were updated to assert the new default and
explicit compatibility behavior, including no repeated notification. They were
not deleted. Existing bars/torus/sphere IDs/names and `orb` alias remain compatible;
the new mode persists as `waveform`. FFT analyzer and torus/sphere mathematics
are unchanged. The existing track timeline still records band-derived loudness.

| Check | Result |
| --- | --- |
| `go test -json -count=1 ./...` | PASS: 402 tests/subtests, 4 opt-in skips, 0 failures; all 6 packages pass |
| `go test -race -json -count=1 ./...` | PASS: same counts, no race reported |
| `go vet ./...` | PASS |
| `CGO_ENABLED=1 go build -o /tmp/piskin-phase4 .` | PASS, native Linux ARM64 |
| `CGO_ENABLED=0 go build -o /tmp/piskin-phase4-nocgo .` | PASS, native Linux ARM64 |
| Windows amd64 and macOS arm64 nocgo cross-builds | PASS; compilation only, not native runtime/capture verification |
| `AMTUI_TEST_PULSE_FIXTURE=1 go test -race -v -count=3 -timeout=35s ./visualizer -run '^TestPulseIsolatedLifecycle$'` | PASS, all three isolated runs |
| Live source/service and root UI checks below | PASS, native ARM64 PipeWire capture |
| `git diff --check` and full diff review | PASS |

The four ordinary-suite skips are the explicit live source/service, live PCM UI,
isolated server lifecycle and disposable headed-browser checks. The first three
were executed separately for this phase; the disposable browser login test was
not repeated. No new dependencies were added. No profiles, credentials, binary,
audio recording or temporary probe is committed.

Deterministic tests cover real input-domain properties using generated PCM:
zero silence with valid live metadata; whole-cycle sine amplitude/RMS; positive
and negative impulses at window/bin boundaries; unequal stereo and anti-phase
signals with signed channel separation; irregular sample reads that split stereo
frames; startup partial windows; wrapping beyond 4096 frames; chronological
min/max; chunk-independent PCM and unchanged FFT output; and value ownership
across Latest copies and capture-buffer reuse. Rendering checks cover signed L/R,
window/format metadata, impulse retention at narrow widths, small/zero dimensions,
mode persistence and the freshness boundary. Stale capture clears PCM/peaks and
history eligibility; a new frame restores PCM. Terminal failure clears prior
signal quietly, and reopening with a different source/format and real silence
restores LIVE with new metadata. Demo remains explicit and cannot replace
starting, stale or live capture, even when enabled.

### Native live PCM and UI repeat

The existing session identified PulseAudio on PipeWire 1.6.9. The application's
actual Go backend selected `audio_effect.j293-convolver.monitor`, negotiated
float32 at 48,000 Hz / 2 channels, and reported PIPEWIRE. Both Source and Service
were tested with the race detector; generated PCM was not injected as a source.
The service smoke test now checks waveform range/levels, format, increasing
complete-frame counts, chronological timestamps and finite extrema as well as
its existing FFT/source checks. Only aggregate levels/times survive the tests.

Controlled playback used the existing public `engine.Engine` methods through a
temporary probe outside the checkout: official headed Chromium/MusicKit and the
already authorized persistent session. Catalog `636968288`, “Get Lucky” radio
edit, reported a 4:08 duration, playing without initialization/player errors.
This repeats PCM verification, not the earlier full two-minute subscription or
user-confirmed audibility gate. No account identifiers or browser data were
exported, and DRM, entitlement handling and desktop configuration were unchanged.

Times below are TRT (UTC+03:00), local read completion rather than hardware time.

| Controlled state | Actual capture result |
| --- | --- |
| Initially idle, 22:53:32–38 | Fresh Source/Service reads, all zero PCM; 91 observed service frames, 141120 complete frames received, zero envelope RMS/peak and FFT levels |
| Playing, 22:55:15–21 | Source RMS 0.114275, peak 0.498163; Service 96 observed frames / 86 spectrum changes, 4096-frame / 128-point envelope, max channel-window RMS 0.230959 and peak 0.500557 |
| Paused at 1:08; capture 22:56:47–53 | Source/Service PCM, envelope RMS/peak and FFT all zero; 91 observed service frames, still fresh/live |
| Resumed, 22:58:08–14, final implementation | Source RMS 0.121734, peak 0.500018; Service 93 observed frames / 86 spectrum changes, 142560 complete frames received, max channel-window RMS 0.208644 and peak 0.508547 |

Live commands:

```sh
AMTUI_TEST_PULSE=1 go test -race -v -count=1 -timeout=25s ./visualizer -run '^TestPulseLiveSmoke$'
AMTUI_TEST_PULSE=1 AMTUI_TEST_PULSE_REQUIRE_SIGNAL=1 go test -race -v -count=1 -timeout=25s . ./visualizer -run '^(TestPulseLiveSmoke|TestPCMLiveUI)$'
```

`TestPCMLiveUI` passed on actual silent and playing capture through Service →
model → View, all four modes at 40×12, 80×24 and 120×32. Frame bounds, LIVE/backend
labels and PCM rendering passed. During resumed playback its first owned
1440-frame snapshot had L/R peaks 0.156711 / 0.128368 and RMS 0.0516543 / 0.0534941.
The pause run had zero peaks/RMS and remained live. This is rendered-frame
verification; interactive terminal input was not repeated. The probe paused
playback and closed only its owned browser after checking state at 1:47.

The isolated PipeWire fixture passed three times under race: reopening after a
default-sink change selected the new monitor, removing that monitor moved capture
to the surviving sink, server loss surfaced EOF, and a fresh server/source
restored capture. Only fixture-owned daemons were stopped. The user's audio
server was never restarted. Actual physical device hotplug, native macOS tap,
Windows runtime and resource/performance measurements remain phase 5 work.

## Resource optimization — phase 5 (2026-11-11)

Baseline: `1e2b3be8bc1e098eabee58a9e9707dfa3ffe415d`. Optimized source is the
resource optimization commit containing this record. Both were measured on this
same Apple MacBook Pro (13-inch, M1, 2020), running Linux ARM64
`7.2.11-14-1-ARCH`, Go `1.26.11`, with `GOMAXPROCS=4` and the existing `schedutil`
governor. No desktop settings, dependencies, browser profile or DRM changed.

### Same-target performance comparison

`scripts/measure-performance.py` checks out the baseline in a disposable detached
worktree, copies the exact same `performance_test.go` into it, and compiles both
test executables before measuring them sequentially. Harness SHA-256:
`e52074261f989367bf08f45892da4ce0f52d131dd9a4f7dcb5908fc9f4981460`.
The workload uses synthetic metadata/spectrum and a deterministic 256px color
image, 120×40 terminal cells, TrueColor, warmed caches, real Update/View code,
five one-second benchmark samples per case. Browser/CDP, network, capture/FFT,
and terminal I/O are excluded. This measures target UI cost, not whole-player
CPU or an audible subscription session. No other build/test jobs ran during
the comparison. The governor and other ordinary desktop processes were not
controlled; five samples do not establish statistical significance.

Median Update/View results (forced equal tick counts, so cadence savings are
separate):

| Workload | Before ms/frame | After ms/frame | Before → after allocations/frame |
| --- | ---: | ---: | ---: |
| Focus playing | 24.76 | 11.95 | 68,816 → 6,355 |
| Lyrics playing | 22.31 | 9.27 | 68,841 → 6,383 |
| Visualizer playing | 10.06 | 10.48 | 16,394 → 16,052 |
| Library playing | 3.37 | 2.31 | 4,709 → 4,387 |
| Focus idle | 24.51 | 9.64 | 68,788 → 6,325 |
| Visualizer reduced motion | 10.62 | 8.41 | 16,380 → 16,065 |
| Visualizer covered by help | 0.98 | 0.75 | 1,596 → 1,565 |
| Shared layout only | 0.0101 | 0.0036 | 51 → 14 |

Focus/Lyrics time decreased 52%/58%, with about 91% fewer allocations. Median
allocated bytes/frame decreased from 5.57/5.82 MB to 3.70/3.74 MB respectively.
The active visualizer median was 4.2% slower; its sample ranges overlap, so no
per-frame improvement is claimed for that uncached drawing path.

A separate real-timer loop ran each workload for roughly three seconds using
the actual production timer factory. Each iteration waits for its timer, calls
Update and View, and discards effects; it is not the full Bubble Tea scheduler.
Observed cadence includes synchronous rendering overhead and is below the
configured maximum:

| Workload | Before observed Hz | After observed Hz |
| --- | ---: | ---: |
| Focus playing | 17.98 | 3.80 |
| Focus idle | 16.37 | 3.82 |
| Visualizer playing | 19.93 | 17.99 |
| Visualizer reduced motion | 25.00 | 7.93 |
| Blurred visualizer | 18.54 | 3.99 |

Across these five timed cases, process user+system CPU was **4.957 → 1.888 s**
(62% lower); peak RSS was **45,712 → 44,560 KiB** (roughly unchanged). Blurred
allocation fell from 110.10 MB to 85.7 KB per three-second run. RSS is the whole
test executable's high-water mark; retained-heap snapshots include uncollected
objects and are not a leak test. Cache bounds are tested directly: at most 16
artwork variants and 32,768 cells, one geometry/header entry and one last frame.
Existing waveform storage remains fixed at 4096 stereo PCM frames, 128 envelope
points and 240 track loudness buckets, regardless of track duration.

Raw records: [before benchmark](docs/performance/before-bench.txt),
[after benchmark](docs/performance/after-bench.txt),
[before timed loop](docs/performance/before-cadence.txt),
[after timed loop](docs/performance/after-cadence.txt).
Reproduce on an otherwise idle Linux target:

```sh
GO=/path/to/go python3 scripts/measure-performance.py 1e2b3be /tmp/piskin-performance
```

### Verification and acceptance

| Check | Result on this Linux ARM64 host |
| --- | --- |
| Unchanged baseline `go test ./...` | PASS, all six packages |
| `go mod verify`, `go vet ./...` | PASS |
| Final `go test -json -count=1 ./...` | 408 PASS, 0 FAIL, 6 explicit opt-in SKIP; all six packages pass |
| Final `go test -race -json -count=1 ./...` | Same counts; no race reported |
| Native cgo and nocgo builds | PASS |
| Linux amd64/arm64 and Windows amd64/arm64 nocgo release builds | PASS |
| Darwin amd64/arm64 nocgo cross-builds | PASS, explicitly without live capture; not macOS release substitutes |
| PTY acceptance using race-instrumented test executable | PASS: actual keyboard, SGR mouse, focus-report and SIGWINCH input through Bubble Tea |
| Real Linux Source/Service/UI capture repeat under race | PASS with fresh silence, not a new playing/audibility claim |
| `git diff --check` | PASS |

New regression checks compare cached cells to the original renderer at TrueColor,
256-color and ASCII profiles, exercise resize/track churn against cache bounds,
compare cached geometry/header to uncached results across screens/overlays and
palette changes, and verify continued live history behind hidden layers. Timing
checks cover paused/reduced/hidden cadence, frozen shape orientation, no reduced
pulse, blur/focus refresh, notice expiry, progress/scrobble elapsed accounting,
and capped suspend gaps. The earlier deterministic PCM/stereo/ownership tests
remain in place. No waveform implementation was replaced just to re-establish
an already fixed storage bound.

The PTY driver uses fixture playback without browser/account access. It visits
all four screens by keyboard/mouse, pauses by keyboard and resumes by mouse,
seeks via the drawn progress bar, opens/closes help and queue, sends terminal
blur/focus, resizes below the minimum and back, and quits. Existing navigation,
search, EQ, account, scrolling, overlay isolation and geometry acceptance tests
also pass. This is automated terminal acceptance, not a new manual Ghostty or
native Windows desktop observation.

```sh
go test -race -c -o /tmp/piskin-terminal.test .
python3 scripts/terminal-acceptance.py /tmp/piskin-terminal.test
AMTUI_TEST_PULSE=1 go test -race -v -count=1 -timeout=25s . ./visualizer \
  -run '^(TestPulseLiveSmoke|TestPCMLiveUI)$'
```

The live repeat at 01:27 TRT selected PIPEWIRE, float32/48,000 Hz/stereo.
Source and Service each read 288,000 finite samples over three seconds; Service
observed 89 fresh frames and a 4096-frame/128-point window. RMS, peaks and FFT
were zero throughout. UI received a fresh 1440-frame envelope and passed all
four modes at 40×12, 80×24 and 120×32. No playback was started for this repeat.
The earlier playing/paused/resumed phase 4 evidence remains historical evidence.

### Dependencies, migration, packaging and outstanding gates

`go.mod`, `go.sum` and upstream MIT LICENSE are unchanged. The review checked
license material for all 10 direct and 19 indirect pinned requirements, including
Windows-only modules. Root module licenses are MIT/BSD; Gonum also supplies
incorporated-algorithm notices. `THIRD_PARTY_NOTICES.txt` preserves supplied
licenses, authors/patents and those supplemental notices plus the Go runtime
license. The pinned `go-localereader` has an MIT/author declaration in README
and no standalone license file; that original README is included. This is a
source/notice review, not a vulnerability audit or a license for external CDMs.
No browser, DRM components, fonts or music content are bundled.

Migration/configuration behavior is documented in
[docs/RELEASE_NOTES.md](docs/RELEASE_NOTES.md). Packaging includes the original
LICENSE, dependency notices, README, verification/design documents, example
configuration and migration notes. `make dist VERSION=phase5-dev` built all four
Linux/Windows archives and SHA256SUMS on Linux; macOS release targets were skipped
explicitly. All four archive hashes were independently recomputed, expected
binary/document/configuration/notice entries inspected, and their LICENSE bytes
compared to the original: PASS. These are development artifacts under ignored
`dist/`, not a published release. No release tag was created.

**Still required before phase 5 can be declared fully complete:** native macOS
permissioned playing/paused/resumed capture. This Linux session cannot execute
CoreAudio directly. The macOS development workflow runs native cgo tests,
race/vet/builds, the Mach-O deployment check and six-target packaging; the Windows
workflow retains its browser/installer acceptance. Hosted native unit tests alone
cannot establish permissioned live capture. The strengthened macOS live smoke
tests now fail on zero received PCM and support `REQUIRE_SIGNAL=1`; the new live
Service check requires a fresh owned PCM envelope. Both remain opt-in, with
commands in the release notes. Unsupported capture remains honestly unavailable.

The first native CI attempt was cancelled before any steps ran: GitHub rejected
`macos-14-arm64-us-default` as unsupported. GitHub's
[runner image inventory](https://github.com/actions/runner-images/blob/main/README.md)
records macOS 14 retirement on November 2, 2026. Development verification now
uses the supported `macos-15` and `macos-15-intel` labels for native ARM64/Intel
checks; release packaging uses `macos-15`. The deployment target remains 14.2.
The cancelled attempt is not a test pass.

On source `ae54bd5`, [native Windows CI](https://github.com/theonheptagrammaton/piskinmusic/actions/runs/42915361625)
passed vet, all tests, browser window integration and installation/aliases through
Windows PowerShell 5.1 and PowerShell 7 (including repeated install/PATH behavior).
System capture remains unsupported on Windows.

The [native macOS run](https://github.com/theonheptagrammaton/piskinmusic/actions/runs/42915361607)
passed `make verify` on both ARM64 and Intel: tests, race, vet, cgo/nocgo builds
and `vtool` output confirming **minos 14.2 / SDK 15.5**. These are native
conversion/ring/lifecycle checks; the permissioned live tests were not enabled.
Six-target packaging then failed in notice generation because the fresh macOS
module cache lacked Windows-only `coninput`. The notices prerequisite now
downloads every pinned requirement before enumerating license files. The failed
packaging attempt is not counted as a release packaging pass.

### Final native and packaging results

Source `dca2f82` passed the corrected [macOS matrix and six-target packaging
workflow](https://github.com/theonheptagrammaton/piskinmusic/actions/runs/42916388054)
and [native Windows workflow](https://github.com/theonheptagrammaton/piskinmusic/actions/runs/42916388133).
ARM64/Intel both passed `make verify`; ARM64 then built native cgo Darwin
arm64/amd64 and pure-Go Linux/Windows arm64/amd64 archives. The run uploaded
`amtui-development-packages` (six archives plus SHA256SUMS) without publishing
a release. A separate local test reproduced every notice byte-for-byte from an
initially empty module cache before the packaging fix was committed. Module
versions/checksums and the original application LICENSE remain unchanged.

Local `dist/` holds the four Linux/Windows development archives. Their checksums,
contents and notice/license/config/document bytes were verified against source;
the packaged ARM64 executable returned `PISKIN MUSIC (amtui) phase5-dev linux/arm64`.
The six native-run artifacts were also downloaded and independently inspected:
all SHA-256 hashes, required entries and original LICENSE bytes match; ELF/PE/
Mach-O CPU headers match each named target; both native cgo Mach-O executables
declare **minos 14.2**. The downloaded bundle is available in this session at
`/tmp/piskin-phase5-native-dist/` and from the linked successful run. Permissioned
macOS live capture remains **NOT VERIFIED**, and phase 5 is not marked fully
complete. The tested production Go source is identical across `5429d5b`,
`ae54bd5` and `dca2f82`; the latter changes correct CI/package prerequisites and
record evidence.

Branch verification identified the configured user-owned fork
`theonheptagrammaton/piskinmusic`, ADMIN push permission, `main` as the default,
and no applicable rules on `development/performance`. Pushes target only
`origin/development/performance`, which now tracks that unprotected branch;
remote `main` remains at `1e2b3be`. No upstream push, tag or public release was
created. The final documentation-only receipt does not change tested Go source.

## Visual polish — 2026-11-11

Implemented the approved appearance changes: right-aligned toolbar and spaced
navigation, 28–38-column Library sidebar, full-width listening screens with an
explicit queue overlay, centered Focus artwork/metadata, quiet section rules,
aligned list durations, and a borderless transport with clickable previous/next
controls. Lyrics use more line spacing when height permits and center short
blocks; their active line has a textual marker. Artwork retains its colors and
the default shell remains monochrome. Playback/capture implementations and
dependencies are unchanged.

Verified locally on Linux ARM64 with `go1.27.2-X:nodwarf5`:

- Baseline and final `go test ./...`: PASS.
- Final `go test -race ./...` and `go vet ./...`: PASS.
- `go build -o amtui .` and `CGO_ENABLED=0 go build ./...`: PASS.
- Compiled-test PTY acceptance through `scripts/terminal-acceptance.py`: PASS
  for keyboard, mouse play/seek/navigation, blocking help, focus and resize.
  Updated the scripted Lyrics click for the new navigation spacing.
- Added regressions for actual rendered toolbar/navigation/transport targets
  across 40–180 columns, including the full-toolbar breakpoint; content/sidebar
  proportions; Focus centering with/without artwork and long Unicode metadata;
  preserved queue durations and help hints; centered/spaced lyrics and their
  no-color current-line marker. Existing resize/scroll/overlay checks pass.
- Inspected images rendered from production `View()` output at 120×35, 80×24
  and 40×12 using synthetic metadata/artwork. The visualizer preview was
  explicitly labeled DEMO/SIMULATED. These are fixture previews, not device
  screenshots or evidence of fresh live capture.

No new subscription playback, audible-output or permissioned capture pass is
claimed for this appearance-only change. Previous platform runtime evidence and
outstanding macOS permissioned-capture checks retain their existing status.
