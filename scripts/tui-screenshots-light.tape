# VHS Tape Script for mjr.wtf TUI Light Theme Screenshots
# Generates PNG screenshots under docs-site/public/images/tui/screenshots/.

Output docs-site/public/images/tui/screenshots/.tui-screenshots-light.gif

Set FontSize 16
Set Width 1400
Set Height 800
Set Padding 20
Set Theme "Catppuccin Latte"
Set PlaybackSpeed 1.0

Type "export MJR_BASE_URL=http://localhost:8080"
Enter
Sleep 200ms

# Empty state (new token with no URLs)
Type "export MJR_TOKEN=empty-token"
Enter
Sleep 200ms
Type "./bin/mjr tui"
Enter
Sleep 2s
Screenshot docs-site/public/images/tui/screenshots/tui-empty-light.png
Type "q"
Sleep 500ms

# Browsing + create
Type "export MJR_TOKEN=demo-token"
Enter
Sleep 200ms
Type "./bin/mjr tui"
Enter
Sleep 2s
Screenshot docs-site/public/images/tui/screenshots/tui-browsing-light.png

Type "c"
Sleep 1s
Screenshot docs-site/public/images/tui/screenshots/tui-create-light.png

Type "q"
Sleep 500ms
