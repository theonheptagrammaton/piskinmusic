# PISKIN MUSIC — interaction and visual design

Implementation boundaries are in [ARCHITECTURE.md](ARCHITECTURE.md), sequencing
in [PLAN.md](PLAN.md), and acceptance evidence in [TESTING.md](TESTING.md).

## Direction

A native terminal music instrument: fast navigation, readable metadata, precise
focus, generous empty space where dimensions permit. Use terminal cells and
normal list interactions rather than web-style cards, decorative dashboards or
large ASCII branding. The monochrome shell and four-screen phase is implemented; PCM waveform,
explicit demo policy and resource optimization remain later phases.

The shell uses black and white. Neutral grayscale may express subdued text,
separators and depth, with no chromatic accents. Album artwork keeps its original
colors and must not recolor the surrounding UI. Use an explicit black background
where the terminal supports it; retain usable text and focus in reduced-color
terminals. Selection must remain identifiable without color.

## Central theme and reusable components

Extend `theme.go`, rather than spreading escape sequences through views.
Semantic roles: background black; primary text white; secondary text neutral
gray; focused row inverse black-on-white; selected but unfocused row a leading
marker; quiet separator gray; status text with an explicit word or symbol.
Never communicate error, live state or keyboard focus through color alone.

Shared components: screen navigation, focusable list, metadata block, artwork,
transport/progress, source-status line, empty/loading/error state, overlay frame,
key hints. Reuse the existing panel helper only where a boundary aids navigation;
avoid nesting borders. Components consume cell dimensions and return clipped
output. The layout owns geometry for both rendering and mouse input.

The existing palettes remain available as explicit legacy choices for
compatibility. The PISKIN default and its artwork behavior are fixed monochrome;
changing that default and disabling artwork-driven shell accents are intentional
product changes. Existing saved preferences are not silently rewritten.

## Four primary screens

| Screen | Primary content | Supporting interaction |
| --- | --- | --- |
| Library | User albums/playlists, recent items, catalog search access | Reuse current library/search engine capabilities. Queue remains accessible; do not imply unsupported library editing. |
| Focus | Current cover, title/artist/album, transport and restrained progress | Queue and playback settings available without changing playback context. |
| Visualizer | Spectrum, PCM Waveform, Torus, Sphere | Persistent mode name, source/capture status, compact now-playing and transport. |
| Lyrics | Readable synchronized text, current line, optional artwork | Show plain lyrics when synchronization is unavailable; clear missing/loading states. |

Screen changes preserve playback, queue, search results, selection and scroll
positions. Overlays are contextual layers, not additional primary screens.
EQ, account/sign-out, help, theme selection, and Last.fm remain reachable.

## Navigation proposal

- `1` Library, `2` Focus, `3` Visualizer, `4` Lyrics, when no text field or
  blocking overlay owns input. Display the active screen in the navigation line.
- Preserve `Space`, `n/p`, `s/r`, `/`, `R`, `q`, `Ctrl+C` and existing transport
  commands. Preserve `Q/P` EQ controls, `a` account behavior and contextual
  search queue actions. Consult `help.go` before assigning any additional keys.
- `Tab` cycles visible focus regions on the active screen; add `Shift+Tab`
  to reverse. Never focus a hidden panel. Maintain existing queue/recent/transport
  traversal when those regions are displayed together.
- `j/k`, arrow keys, `Enter`, and `Esc` retain context-appropriate behavior.
  Search typing must receive characters before global screen shortcuts.
- `v` cycles visualization modes without leaving the active playback context.
  Keep persisted bars/torus/sphere values compatible; add a distinct waveform ID.
- Help lists all actions. Every mouse action has a keyboard equivalent.

Clicks select/activate list rows, screen labels and transport targets; scrolling
moves the list under the pointer. Preserve existing click-to-play and seek
semantics. Clip hit targets to visible cells, and prevent overlay click-through.
Render and test keyboard focus separately from playback/selection markers.

## Responsive layout

All sizes are terminal cells, not pixels. Proposed breakpoints are starting
points to validate in Ghostty, not assumptions about the current layout.

| Size | Behavior |
| --- | --- |
| At least 120 columns and 32 rows | Optional queue/support column; artwork and main content side by side where useful. |
| 80–119 columns, at least 24 rows | Primary content takes priority; supporting panels stack or become overlays; compact transport. |
| 40–79 columns or limited height | Single main region, concise navigation and transport; omit artwork before truncating essential controls/metadata. |
| Below 40 columns or 12 rows | Minimal now-playing/status and resize hint, with pause/quit/help still working. No invalid dimensions or offscreen hit targets. |

Test resize while search, help, account and EQ overlays are open. Clip long
Unicode titles using display-cell width; avoid byte slicing and line overflow.
When artwork would crowd lyrics, hide it using the existing width policy.

## Artwork and motion

Start with existing original-color half-block artwork. This is a portable hybrid
of text and raster-derived cells. A later terminal-image protocol enhancement
must have a half-block/text fallback, cleanup on resize/exit and no mandatory
new renderer dependency. Never treat browser canvas as the terminal UI.

Motion comes from audio and subtle focus/track transitions. No screen slides,
flashing borders, compulsory spinners or extra browser work. Offer reduced
motion: static torus/sphere orientation, no pulse, immediate screen changes.
Spectrum/waveform may still show real data; paused playback should be quiet.
Target at most 30 visual frames per second, lowering idle cadence after profiling.

## Visualization truthfulness

- Spectrum: preserve the 32-band analyzer and peak behavior.
- PCM Waveform: show a short, bounded time-domain signal from captured samples.
  A min/max envelope per display column may preserve transients. Retain channel
  information or energy-aware combination so anti-phase stereo does not disappear.
- Torus/Sphere: preserve existing geometry and beat response, rendered in neutral
  tones with modest motion. They share the same real audio analysis.
- Show `LIVE · system output` and the verified backend only for fresh capture.
  System output includes other applications; it is not an isolated track feed.
- Show starting, stale or unavailable explicitly. Quiet failure is the default
  proposed behavior; any demo is explicitly enabled and continuously labeled.
- The transport timeline's existing loudness history is distinct from a PCM
  waveform. Do not invent unplayed history or fill a track with generated samples.

## Acceptance

All upstream operations remain reachable with keyboard and mouse. A monochrome
screen remains understandable without artwork or animation. Track/error state
is readable at narrow sizes. Visualizer source labels agree with backend data
and freshness. Album art retains original colors. Frame rendering performs no
network, browser, capture or filesystem effects.
