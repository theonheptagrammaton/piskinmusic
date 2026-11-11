# PISKIN MUSIC — interaction and visual design

Implementation boundaries are in [ARCHITECTURE.md](ARCHITECTURE.md), sequencing
in [PLAN.md](PLAN.md), and verification evidence in [TESTING.md](TESTING.md).

## Direction

One terminal listening dashboard. Library, Queue, Cover/Now Playing, Visualizer
and Lyrics stay visible simultaneously, including in small windows. The user
confirmed this replaces the earlier four-screen monochrome direction.

The base is black and white, with grayscale secondary text and quiet rules.
Headings, selected items and all visualization modes use the seven-color
PISKIN rainbow. Album artwork keeps its original colors. Text markers retain
focus, selection and playback meaning when color is unavailable.

## Shared rendering and geometry

`theme.go` owns neutral tokens, the brand spectrum and text styling. Existing
named palettes remain available; PISKIN keeps a consistent brand spectrum and
ignores individual color overrides. Saved choices are honored.

`layout.go` owns every panel, list, category tab, recent tile and transport
rectangle. `dashboard.go` consumes those rectangles; mouse input uses the same
content geometry. Dashboard sections reserve one heading row and one cell of
horizontal inset, leaving room for actual content at the minimum size.

Artwork rendering stays cached and bounded. Rendering performs no network,
audio, browser or filesystem effects. Original half-block cover colors must
survive surrounding styles. A missing cover has a visible placeholder; the
explicit artwork-disabled setting is respected.

## Layout

| Terminal size | Arrangement |
| --- | --- |
| At least 100 columns | Library and Queue on the left; Cover above Visualizer in the middle; Lyrics on the right. |
| 40–99 columns | Library above Queue on the left; Cover, Visualizer and Lyrics stacked on the right. |
| At least 100 columns and 35 rows | Also show recent cover tiles below the browsing lists when tiles fit. |
| Below 40 columns or 12 rows | Minimal now-playing/status and resize hint; pause, quit and help remain available. |

At 40×12, each of the five regions retains actual content: a library row, queue
rows, a cover thumbnail, visualization/status and a lyric line. Short sections
use smaller artwork and fewer visible rows; they do not become separate pages.
Library category names shorten to A/P/R when the panel cannot fit full labels.
Long Unicode titles clip by display cells, and list selection scrolls into view.
The bottom transport and its seek geometry remain shared across sizes.

## Navigation and overlays

`1` Library, `2` Cover, `3` Visualizer and `4` Lyrics highlight dashboard regions
without hiding the others or scheduling playback effects. The internal screen
identifier is retained for shortcut compatibility and per-highlight focus.
`Tab` / `Shift+Tab` traverse the visible queue, optional recent grid, library
and transport. Hidden overlay content never receives input.

`[` / `]` change the library category anywhere on the dashboard. `j/k`, arrows,
`Enter`, Space, n/p, seek, volume, shuffle and repeat retain existing behavior.
`v` cycles bars, torus, sphere and waveform; `t` cycles saved named palettes.
Search typing owns its characters before dashboard shortcuts.

`c` expands the queue over the main area; `c` / `Esc` restore the dashboard.
EQ (`Q`), search (`/`), account (`a`) and help (`?`) remain available. Overlays
retain keyboard/mouse controls and close behavior. Search append/play-next,
EQ presets, sign-out, Last.fm and MPRIS remain reachable.

Clicks on library/queue rows and recent covers play the item actually drawn.
The progress bar seeks, transport controls handle play and track changes, and
mouse wheels move the list under the pointer or adjust transport volume.
Whitespace, panel gaps and hidden rows do not activate playback.

## Audio visuals and motion

Bars use the spectrum from left to right; torus and sphere use the same spatial
rainbow with existing geometry and beat response. PCM traces use captured signed
channel envelopes; their sample format and channel identity remain visible when
space permits. The transport timeline remains band-derived loudness, not PCM.

Fresh capture is LIVE, opening waits for PCM, stale frames clear traces, and
unavailable capture stays quiet. Only explicit demo configuration produces
simulated bands, continuously labeled DEMO / SIMULATED even at 40×12. Demo
never creates PCM or history. System output may include other applications.

Visualization is visible for every dashboard highlight. Normal playing data
updates target at most 30 Hz. Reduced motion freezes shape orientation, disables
the bass pulse and caps updates at 10 Hz. Covered, blurred and idle UI uses the
existing lower cadence; capture and analysis continue independently.

## Acceptance

All five sections retain content from 40×12 upward and across highlight changes.
Panel dimensions, scrolling and pointer geometry must agree during resize.
The rainbow appears in headings, selected items and all four visualizers;
neutral body text and original-color artwork remain legible. Existing named
palette selection and explicit artwork settings remain compatible.

Run the complete Go suite, race check, vet, native build and PTY input acceptance.
These checks cover rendering and interaction; they do not replace real audible
subscription playback or permissioned hardware capture verification.
