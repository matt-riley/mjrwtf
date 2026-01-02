# VHS Tape Script for mjr.wtf TUI Dark Theme Screenshots
# Generates PNG screenshots under docs/screenshots/.

Output docs/screenshots/.tui-screenshots-dark.gif

Set FontSize 16
Set Width 1400
Set Height 800
Set Padding 20
Set Theme "Catppuccin Mocha"
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
Screenshot docs/screenshots/tui-empty-dark.png
Type "q"
Sleep 500ms

# Browsing + create
Type "export MJR_TOKEN=demo-token"
Enter
Sleep 200ms
Type "./bin/mjr tui"
Enter
Sleep 2s
Screenshot docs/screenshots/tui-browsing-dark.png

Type "c"
Sleep 1s
Screenshot docs/screenshots/tui-create-dark.png

Type "https://github.com/matt-riley/mjrwtf"
Sleep 300ms
Tab
Sleep 300ms
Type "demo"
Sleep 300ms
Enter
Sleep 2s

# Exit to generate some clicks, then re-open for analytics
Type "q"
Sleep 500ms
Type "for i in 1 2 3; do curl -s -o /dev/null http://localhost:8080/demo; done"
Enter
Sleep 500ms

Type "./bin/mjr tui"
Enter
Sleep 2s
Type "a"
Sleep 2s
Screenshot docs/screenshots/tui-analytics-dark.png
Type "b"
Sleep 1s

# Delete confirmation
Type "d"
Sleep 1s
Screenshot docs/screenshots/tui-delete-confirmation-dark.png
Type "n"
Sleep 500ms
Type "q"
Sleep 500ms
