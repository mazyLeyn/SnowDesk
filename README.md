# ❄️ SnowDesk

Falling snow, snowmen and snow-capped windows on your Windows desktop.
A tiny, dependency-free, open-source winter overlay.

![preview](docs/preview.png)

*(Türkçe için: [README.tr.md](README.tr.md))*

## Features

- Soft falling snow with wind and depth (big flakes fall faster)
- **Snow piles up on top of every window** and the taskbar; drag a window and the snow shakes off in a puff, and is hidden correctly when another window is in front
- Optional animated snowmen on your taskbar (off by default, enable in the tray menu)
- **Click-through**: the overlay never steals focus or mouse clicks
- Hides itself automatically when a fullscreen game/video is in front
- Tray icon: pause, snowmen on/off, intensity, clear snow, *Start with Windows*
- One ~2 MB `.exe`, no installer, no runtime, no admin rights, no network access

## Install

Download `snowdesk-windows-amd64.exe` (or `-arm64`) from [Releases](../../releases) and run it.
Right-click the snowflake tray icon for options. Windows 10 / 11.

## Options

Everything is in the tray menu. Power users can also use command-line flags
or edit `%AppData%\SnowDesk\config.json` (settings changed in the menu are saved there).

| Flag | Default | Meaning |
|---|---|---|
| `-density` | 120 | flakes per megapixel (60 light, 260 blizzard) |
| `-wind` | 18 | wind strength |
| `-snowmen` | 0 | number of snowmen (off by default) |
| `-snowman-scale` | 1.3 | snowman size |
| `-piles` | true | snow piles up on windows |
| `-pile-rate` | 1 | how fast it piles up |
| `-max-pile` | 10 | max pile height (px) |
| `-fps` | 30 | frame rate |
| `-all-monitors` | false | cover all monitors (uses more CPU) |

## Build from source

Needs Go 1.21+. Cross-compiles from any OS.

```sh
make            # dist/snowdesk-windows-{amd64,arm64}.exe
make test       # unit tests + go vet
make preview    # render docs/preview.png without Windows
```

## How it works

- `snow/` – pure Go simulation and software renderer (flakes, piles, snowmen). No OS code, fully unit-tested, runs anywhere.
- `cmd/snowdesk/` – Windows glue using only `syscall`: a layered `WS_EX_TRANSPARENT` window updated with `UpdateLayeredWindow`, `EnumWindows` + DWM frame bounds to find window title bars, `Shell_NotifyIcon` for the tray.
- `cmd/preview/` – renders the scene to a PNG for screenshots.

## Contributing

Issues and PRs welcome: new decorations (lights, a Santa sleigh, other seasons), more monitors support, themes…

## License

MIT
