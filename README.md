# PISKIN MUSIC — Apple Music in your terminal

PISKIN MUSIC adds a monochrome shell and four screens to the MIT-licensed
[applemusic-tui](https://github.com/k1y0miiii/applemusic-tui) by k1y0miiii.
The executable remains `amtui`; existing aliases, config and authentication
profiles, and MPRIS IDs are preserved. The screenshots below show the upstream
interface and have not yet been refreshed for the four-screen shell.

**English** · [Русский](README.ru.md)

![Go 1.26](https://img.shields.io/badge/Go-1.26-555?style=flat-square&logo=go&logoColor=white)
![Platforms](https://img.shields.io/badge/platforms-macOS%20%C2%B7%20Linux%20%C2%B7%20Windows-555?style=flat-square)
![License: MIT](https://img.shields.io/badge/license-MIT-555?style=flat-square)

`amtui` is a terminal Apple Music player — a full TUI client with catalog
search, a live audio spectrum visualizer and synced lyrics, running on **Linux
as well as macOS**. It drives the official Apple Music web player inside a
hidden Chromium, so playback is the real thing. No DRM hacks: the browser plays
the audio legally, amtui just gives it a terminal face.

<p align="center"><img src="docs/media/demo.gif" alt="amtui — Apple Music in the terminal" width="800"></p>

## Why amtui

Terminal Apple Music clients are usually AppleScript remotes for the macOS
Music.app: Mac-only, and limited to steering an app that has to be running
anyway. amtui talks to Apple Music's **web** player through MusicKit instead,
which changes what is possible:

- **Linux, not just macOS.** The same Go binary runs on both. (Linux support is
  newer — see the note under [Requirements](#requirements).)
- **The full catalog, in the terminal.** Search songs, albums and playlists
  straight from MusicKit — no desktop app in the loop.
- **A real spectrum visualizer.** Actual system-audio PCM through an FFT, not a
  decorative animation.
- **Synced lyrics** from LRCLIB, with the album cover rendered next to them.
- **Your credentials stay out of the terminal.** You sign in inside a real
  browser window, so 2FA and captcha just work.

## Features

- **Full Apple Music catalog** — search songs, albums and playlists, plus your
  recently played, straight from MusicKit.
- **Queue** — jump to any track, append to queue, play next.
- **Recently played** — a cover grid under the queue: your last ten albums,
  playlists and singles as half-block artwork, arrow keys to pick, `↵` to play.
- **Live visualizer** — what is actually playing: system-audio PCM through a
  4096-sample Hann-window FFT into 32 bands (denser at low frequencies), 30 fps.
  CoreAudio process tap on macOS, PipeWire/PulseAudio monitor on Linux. Capture
  failure shows a quiet unavailable panel; the labeled compatibility demo is
  opt-in. Four modes, cycled with `v`: a real **PCM waveform**, Winamp-style
  **bars**, a spinning **torus** whose tube corrugates with the spectrum, and a wireframe **sphere**
  that turns slowly and pumps with the bass.
- **Synced lyrics** — timestamped lines from [LRCLIB](https://lrclib.net),
  no API keys, with the album cover rendered beside them in half-block color.
- **Last.fm scrobbling** — optional, off until you set it up. Counts time
  actually listened, so seeking to the end does not scrobble a track.
- **MPRIS on Linux** — media keys, panel widgets, lock screens and `playerctl`
  drive it like any desktop player. Appears automatically when there is a
  session bus.
- **Themes** — six built-in palettes cycled with `t`, overridable per color in
  a config file, plus an `auto` mode that pulls the accent from the artwork.
- **Transport** — play/pause, next/prev, seek, volume, shuffle, repeat, and a
  progress bar showing recorded band-derived loudness along the track (not PCM).
- **10-band equalizer** — real Web Audio filters on the playback chain, `Q` to
  swap the visualizer for it, presets cycled with `P`, gains persisted across
  runs.
- **Safe sign-in** — you log in inside a real browser window; your Apple ID
  credentials never touch the terminal.

## How it works

Apple Music has no public streaming API, and grabbing the stream would mean
breaking DRM — off the table. Instead, the browser plays the music legally and
amtui remote-controls it:

```
┌──────────────┐   chromedp (CDP)   ┌───────────────────────────┐
│  amtui (Go)  │ ◄────────────────► │ hidden Chromium           │
│  Bubble Tea  │                    │ music.apple.com, signed in│
└──────────────┘                    │ window.MusicKit (JS API)  │
                                    └───────────────────────────┘
```

One Go binary. It spawns a hidden Chromium with a persistent profile in
`~/.config/amtui/chrome` and talks to `window.MusicKit` in page context —
search, queue, play/pause, seek, now playing. No DOM scraping. Audio goes out
through the system mixer (the web player serves AAC 256; no lossless).

<a id="requirements"></a>

## Requirements

- An active **Apple Music subscription**
- **Go 1.26+** (to build from source)
- **Chrome or Chromium**
  - macOS: any recent Chrome/Chromium
  - Linux: a Widevine-capable browser — `google-chrome`, or Chromium with the
    Widevine plugin (e.g. `chromium-widevine` on the AUR)
  - Windows: Google Chrome or Microsoft Edge (auto-detected; `AMTUI_CHROME`
    overrides)
- **macOS 14.2+** (the live visualizer uses a CoreAudio process tap),
  or **Linux** with `pipewire-pulse` (or plain PulseAudio) for live capture

> Linux support is **experimental** — designed for Arch/Ubuntu, currently less
> tested than macOS. On Hyprland the browser window is auto-hidden into a
> special workspace (Wayland forbids offscreen positioning); other Wayland
> compositors may leave the window visible for now.

> Windows is now built and tested in CI, but there is still no system-audio
> capture backend, so the visualizer shows UNAVAILABLE by default. Set
> `visualizer.demo = true` for the legacy simulated animation. Use Windows
> Terminal — the legacy console does not render the TUI correctly. From Git Bash, run it with `winpty amtui`.

## Install

### Prebuilt binaries

Grab the archive for your platform from the
[latest release](https://github.com/k1y0miiii/applemusic-tui/releases/latest) —
macOS (Apple Silicon / Intel), Linux (x86-64 / arm64) and Windows
(x86-64 / arm64):

```sh
tar -xzf amtui-*-linux-amd64.tar.gz
sudo install -m755 amtui-*/amtui /usr/local/bin/amtui
amtui --version
```

Every release ships a `SHA256SUMS` file; verify with `shasum -a 256 -c SHA256SUMS`.

macOS binaries are unsigned, so the first launch needs
`xattr -d com.apple.quarantine amtui` (or right-click → Open).

Windows: unzip the archive and run `amtui.exe` from Windows Terminal or
PowerShell. The binary is unsigned, so SmartScreen may say "Windows protected
your PC" — click More info → Run anyway. If Defender quarantines it (a known
false positive for stripped Go binaries), build from source with `install.ps1`
instead.

### From source

```sh
git clone https://github.com/k1y0miiii/applemusic-tui
cd applemusic-tui
./install.sh
```

The script builds from source and installs `amtui` (plus `applemusic` and
`applemusic-tui` aliases) into `~/.local/bin`, adding it to PATH for
zsh / bash / fish. `--prefix DIR` installs elsewhere, `--no-path` leaves your
shell config alone. If Chrome is missing, the installer offers to install it
(Homebrew on macOS, the official `.deb` or AUR on Linux).

On macOS the live visualizer needs the Xcode Command Line Tools
(`xcode-select --install`) — the installer checks for them.

Prefer doing it by hand? `go build -o amtui .` works too; a `CGO_ENABLED=0`
macOS build has no live capture and shows UNAVAILABLE. Linux capture is pure Go
and works without cgo. The legacy demo remains an explicit opt-in on all platforms.

Run the test suite with `make test`, or `make verify` for the full check.

#### Windows

```powershell
git clone https://github.com/k1y0miiii/applemusic-tui
cd applemusic-tui
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

The script builds `amtui` from source and installs it (plus `applemusic` and
`applemusic-tui` shims) into `%LOCALAPPDATA%\Programs\amtui`, adding that
directory to your user PATH. `-Prefix DIR` installs elsewhere, `-NoPath`
leaves your PATH alone. Needs Go 1.26+ and Git on PATH; no admin rights
required.

> Building all release archives yourself: `make dist`. macOS is the only host
> that produces a complete set — the darwin builds need cgo for the CoreAudio
> visualizer, while the Linux and Windows targets are pure Go and
> cross-compile from anywhere.

## First run

1. Launch `./amtui` — a **visible** browser window opens on music.apple.com.
2. Sign in with your Apple ID. It is a normal browser, so 2FA and captcha
   just work.
3. Done — the window hides and the TUI takes over. The session persists in
   `~/.config/amtui/chrome`, so next launches go straight to the player.


## Keys

### Player

| Key | Action |
| --- | --- |
| `Space` | Play / pause |
| `n` / `p` | Next / previous track |
| `1` / `2` / `3` / `4` | Library / Focus / Visualizer / Lyrics |
| `Tab` / `Shift+Tab` | Next / previous visible focus region |
| `c` / `Esc` | Open queue / close queue |
| `[` / `]` | Library: previous / next category (albums, playlists, recent) |
| `j` `k` / `↓` `↑` | Lists: move selection · Recent grid: move a row · Transport: volume down / up |
| `Enter` | Play the selected library item, queue track or recent cover |
| `←` / `→` | Recent: previous / next cover · Transport: seek −5 s / +5 s |
| `s` | Toggle shuffle |
| `r` | Cycle repeat mode |
| `?` | Show keys; arrows/page keys change help page, other keys close |
| `Q` | Open / close EQ; arrows change band/gain, `P` presets, `0` flat |
| `a` | Account; `Enter` signs out, other keys close |
| `v` | Cycle visualizer: bars → torus → sphere → waveform |
| `t` | Cycle color theme |
| `R` | Reload the web player if it wedges |
| `/` | Open search |
| `q` / `Ctrl+C` | Quit |

### Mouse

Click screen labels to navigate and toolbar labels to open search, queue, EQ,
account or help. Click a library item, queue row, recent cover or search result
to play it; scroll over its list to move selection. Search has category and
queue/end/next buttons. Click the play icon to pause/resume, the progress bar to
seek, or wide transport shuffle/repeat labels to toggle them. The transport wheel
changes volume. EQ has clickable gain/preset/reset/close controls; click a band
to select it. General help commands and their individual keys are clickable.

At 40×12 through narrow sizes the main region takes priority, with `c` providing
a full queue. Library also shows a queue from 70 columns; other screens gain a
queue support column at 120×32. Artwork is omitted when space is limited. Below
40×12 a minimal now-playing/resize view retains keyboard pause, help and quit.
Screen changes preserve playback, selections, search results and per-screen focus.

### Search

| Key | Action |
| --- | --- |
| typing | Edit the query |
| `Enter` | In input: search (empty query — your library) · In list: play selection |
| `Tab` | Next tab: RECENT / SONGS / ALBUMS / PLAYLISTS |
| `↓` `↑` / `j` `k` | Move through results (`↓` from the input dives into the list) |
| `a` / `A` | Add to the end of the queue / play next |
| `Esc` | Close search |

<p align="center"><img src="docs/media/search.png" alt="Search overlay" width="800"></p>
<p align="center"><img src="docs/media/lyrics.png" alt="Queue, live visualizer and synced lyrics" width="800"></p>

## Visualizer

Press `v` to cycle four modes; the choice is remembered in
`~/.config/amtui/vizmode`.

| Mode | What it draws |
| --- | --- |
| `waveform` | Signed PCM min/max envelope of the latest 4096 frames (85.3 ms at 48 kHz), with separate L/R traces, sample rate, channel count and window duration. |
| `bars` | 32-band spectrum with peak-hold markers. The default. |
| `torus` | A spinning donut. Its tube thickness at each angle comes from the band that owns that slice of the ring, so the shape corrugates with the spectrum. |
| `sphere` | A wireframe globe of latitude rings and meridians, shaded by depth. Frequency maps to latitude symmetrically, so the bass swells its waist. |

LIVE means fresh PCM received within 400 ms, including silence. STARTING waits
for the first frame; STALE clears traces until fresh data arrives. Terminal
capture errors show UNAVAILABLE quietly. PIPEWIRE/PULSEAUDIO/COREAUDIO identifies
the capture backend; this is system output, so other applications can contribute.
Freshness is measured from local read completion, not a hardware timestamp.
The waveform preserves per-channel extrema, including impulses and anti-phase
stereo; it clips only the display at full scale. The transport's loudness timeline
continues to use FFT bands and is separate from this PCM mode.

To restore the old simulated fallback, explicitly set this in
`~/.config/amtui/config.toml`:

```toml
[visualizer]
demo = true
```

This enables DEMO · SIMULATED only when capture is unavailable. Starting/stale
capture stays quiet, live capture takes priority, and demo never records history
or manufactures a PCM trace. Omitted, false, or invalid values keep demo off.
Existing bars/torus/sphere mode names (and the old `orb` alias) remain compatible;
waveform is persisted as `waveform`.

The two 3D shapes answer the beat differently. The torus answers with motion: a
bass kick speeds its spin up. The sphere answers with size — it turns at about a
third of that pace and instead swells and contracts on the beat, so the pump is
what you watch rather than the rotation.

Both read the beat as how far the low end rises above its own running average,
not as how loud it is. The bands are normalized over a 63 dB window, so a kick
riding 6 dB above the sustained bass moves the absolute number by a tenth and
across a track the low end sits high and nearly flat — driving anything from it
leaves the shape sitting still.

The bars do neither; they are the spectrum itself. Independent of the mode, the
panel borders pulse with the bass — `visualizer.pulse` turns that off.

Both 3D shapes are plain ASCII over the same 32 bands the bars use, so they cost
no extra audio work and follow the active theme's accent colors.

## MPRIS (Linux)

amtui publishes itself on D-Bus as `org.mpris.MediaPlayer2.amtui`, so the
desktop can drive it without the terminal being focused:

```sh
playerctl -p amtui play-pause
playerctl -p amtui metadata
```

Media keys, panel applets and lock screens pick it up on their own. Play,
pause, next, previous, seek and volume are wired; there is no track list and
nothing to raise, and both are declared as such. It appears whenever a session
bus is present and is silently skipped when there is not — a TTY, a container,
or macOS.

## Last.fm

Scrobbling is opt-in. Create an API account at
[last.fm/api/account/create](https://www.last.fm/api/account/create), put the
key and secret under `[lastfm]` in `~/.config/amtui/config.toml`, then run:

```sh
amtui lastfm-auth
```

It prints a link to approve, and saves the session key to
`~/.config/amtui/lastfm` (mode 600) — the key never has to go in the config
file. Nothing is sent until all three parts are present.

Tracks scrobble after half their length or four minutes, whichever comes first,
and never below 30 seconds — Last.fm's own rules. The clock counts time actually
listened rather than the playback position, so dragging the progress bar to the
end does not count as a play.

## Themes

`piskin` is the fixed black/white/grayscale default, including inverse selection
and an explicit black background. Artwork retains its original colors through
the portable half-block renderer. All six original palettes remain explicit
legacy choices: `apple`, `catppuccin`, `gruvbox`, `nord`, `mono`, and `auto`.
Saved palette names and explicit config choices are honored. An existing color-only
config without a palette name keeps the legacy `apple` base; explicitly select
`piskin` to use the fixed monochrome shell. Press `t` to cycle;
the choice is remembered in
`~/.config/amtui/theme`.

`auto` takes its accent from the dominant color of the current track's album
art. Only the accent changes — background and text stay neutral, so no cover can
make the interface unreadable.

The `piskin` palette ignores individual color overrides to keep its standard
appearance monochrome. To customize colors, select a named legacy palette.
To pick a theme up front or override legacy colors, copy
[`docs/config.example.toml`](docs/config.example.toml) to
`~/.config/amtui/config.toml`. The same file switches the peak markers, the bass
pulse, the loudness timeline and the album art on or off, and opts into demo.

## Configuration

Set `[ui] reduced_motion = true` in `config.toml` to hold shapes still, disable
the bass pulse, and limit active visualization updates to 10 Hz. Idle/hidden UI
ticks run at 4 Hz; capture and live history continue independently. See
[migration and release notes](docs/RELEASE_NOTES.md) for resource policy and
platform validation requirements.

| Environment variable | Effect |
| --- | --- |
| `AMTUI_CHROME` | Path to the Chrome/Chromium binary (overrides auto-detection; also covers Microsoft Edge on Windows) |
| `AMTUI_CONFIG_DIR` | Config directory (default `~/.config/amtui`) |
| `AMTUI_DEBUG` | Run the browser visibly, with verbose logging |

## License

[MIT](LICENSE)
