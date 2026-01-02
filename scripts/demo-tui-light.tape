# VHS Tape Script for mjr.wtf TUI Demo (Light Theme)
# This script records a demonstration of the TUI showing key workflows

Output docs-site/public/images/tui/demo-light.gif

# Terminal settings for optimal GIF output
Set FontSize 16
Set Width 1400
Set Height 800
Set Padding 20
Set Theme "Catppuccin Latte"
Set PlaybackSpeed 1.0
Set LoopOffset 20%

# Note: This demo assumes:
# 1. Server is running at localhost:8080
# 2. MJR_TOKEN is set to "demo-token"
# 3. The TUI binary is built at ./bin/mjr

# Show the command being run
Type "# Start the mjr.wtf TUI (Light Theme)"
Enter
Sleep 500ms

Type "export MJR_BASE_URL=http://localhost:8080"
Enter
Sleep 300ms

Type "export MJR_TOKEN=demo-token"
Enter
Sleep 300ms

Type "./bin/mjr tui"
Enter
Sleep 3s

# Navigate the URL list
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "k"
Sleep 800ms

# Show create URL workflow
Type "c"
Sleep 1.5s

# Fill in the form - original URL
Type "https://github.com/matt-riley/mjrwtf"
Sleep 500ms

# Tab to short code field
Tab
Sleep 500ms

# Enter short code
Type "demo"
Sleep 500ms

# Submit the form
Enter
Sleep 2.5s

# Navigate to the newly created URL (should be at top)
Type "k"
Sleep 500ms

# View analytics
Type "a"
Sleep 2.5s

# Go back to list
Type "b"
Sleep 1.5s

# Show delete confirmation (but cancel)
Type "j"
Sleep 300ms
Type "d"
Sleep 1.5s

# Cancel deletion
Type "n"
Sleep 1s

# Refresh the list
Type "r"
Sleep 2s

# Exit gracefully
Type "q"
Sleep 1s
