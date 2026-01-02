# TUI Demo Recording

This placeholder represents where the TUI demo GIF will be placed.

## Recording Instructions

To generate the actual GIF, run:

```bash
# 1. Install dependencies
sudo apt-get install ffmpeg  # or appropriate package manager
go install github.com/charmbracelet/vhs@latest

# 2. Start the server
docker compose up -d

# 3. Verify server is running
curl http://localhost:8080/health

# 4. Build the TUI
make build-mjr

# 5. Record the demo
vhs scripts/demo-tui.tape

# 6. Verify output
ls -lh docs-site/public/images/tui/demo.gif
```

The VHS tape script at `scripts/demo-tui.tape` demonstrates:
- Starting the TUI
- Navigating the URL list with vim keybindings
- Creating a new short URL
- Viewing analytics
- Delete confirmation workflow
- Refreshing the list
- Graceful exit

## Note

This file serves as a placeholder until the GIF is generated. Once `demo.gif` exists, this file can be removed.
