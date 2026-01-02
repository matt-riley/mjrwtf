# Demo Recording Scripts

This directory contains scripts for generating visual assets for documentation.

## TUI Demo GIF

The `demo-tui.tape` file is a VHS (Video for the Terminal) script that records an animated GIF demonstration of the mjr.wtf TUI.

### Prerequisites

1. **ffmpeg** - Required by VHS for video encoding
   ```bash
   # Ubuntu/Debian
   sudo apt-get install ffmpeg
   
   # macOS
   brew install ffmpeg
   ```

2. **VHS** - Terminal recording tool
   ```bash
   go install github.com/charmbracelet/vhs@latest
   ```

3. **Running server** - The demo requires a live server
   ```bash
   docker compose up -d
   # Verify: curl http://localhost:8080/health
   ```

4. **TUI binary** - Build the mjr CLI
   ```bash
   make build-mjr
   ```

### Recording the Demo (Dark)

```bash
# From repository root
vhs scripts/demo-tui.tape
```

This will generate `docs-site/public/images/tui/demo.gif`.

### Recording the Demo (Light)

```bash
# From repository root
vhs scripts/demo-tui-light.tape
```

This will generate `docs-site/public/images/tui/demo-light.gif`.

## TUI Screenshots (PNG)

Issue #218 tracks capturing static screenshots for documentation.

### Prerequisites

Same as the demo GIF:
- ffmpeg
- VHS (`go install github.com/charmbracelet/vhs@latest`)
- a running server (`docker compose up -d` with `AUTH_TOKENS` set)
- built CLI binary (`make build-mjr`)

### Generate screenshots

```bash
# Run server with tokens used by the tapes
AUTH_TOKENS=demo-token,empty-token docker compose up -d

make build-mjr

vhs scripts/tui-screenshots-dark.tape
vhs scripts/tui-screenshots-light.tape
```

This will generate PNGs in `docs-site/public/images/tui/screenshots/`.

Note: The screenshot tapes also produce small “tour” GIFs:
- `docs-site/public/images/tui/tour-dark.gif`
- `docs-site/public/images/tui/tour-light.gif`

Security note: VHS recordings include whatever you type into the terminal. Never use real tokens when recording/regenerating these assets.

### What the Demo Shows

The recorded demo demonstrates:
- Starting the TUI with environment configuration
- Navigating the URL list using vim-like keybindings (j/k)
- Creating a new short URL with the `c` command
- Viewing analytics for a URL with the `a` command
- Delete confirmation workflow (cancelled with `n`)
- Refreshing the list with `r`
- Graceful exit with `q`

### Customizing the Demo

Edit `demo-tui.tape` to change:
- Terminal dimensions (`Set Width`, `Set Height`)
- Color theme (`Set Theme`)
- Playback speed (`Set PlaybackSpeed`)
- Commands and timing (`Type`, `Sleep`, `Enter`)

See the [VHS documentation](https://github.com/charmbracelet/vhs) for more options.

### File Size Optimization

VHS produces optimized GIFs by default. If further compression is needed:

```bash
# Using gifsicle
gifsicle -O3 --colors 256 docs-site/public/images/tui/demo.gif \
  -o docs-site/public/images/tui/demo.gif
```

Target: Keep the GIF under 1-2MB for reasonable repository size and fast page loads.
