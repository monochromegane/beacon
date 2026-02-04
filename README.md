# beacon

[![Actions Status](https://github.com/monochromegane/beacon/actions/workflows/test.yaml/badge.svg?branch=main)][actions]

[actions]: https://github.com/monochromegane/beacon/actions?workflow=test

beacon is a CLI tool that tracks Claude Code session states within tmux.

## Installation

```bash
go install github.com/monochromegane/beacon@latest
```

## Usage

beacon provides two main commands:

- `beacon emit` - Emits signals from Claude Code hooks (reads JSON from stdin)
- `beacon scan` - Scans tmux for signals and displays status

## Claude Code Hooks Configuration

Add the following hooks to `~/.config/claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [{ "hooks": [{ "type": "command", "command": "beacon emit" }] }],
    "SessionEnd": [{ "hooks": [{ "type": "command", "command": "beacon emit" }] }],
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "beacon emit" }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "beacon emit" }] }],
    "PreToolUse": [{ "hooks": [{ "type": "command", "command": "beacon emit" }] }],
    "Notification": [{ "hooks": [{ "type": "command", "command": "beacon emit" }] }]
  }
}
```

## Tmux Status Line Example

Create a script (e.g., `beacon-status`) that uses a Go template to display colored status indicators:

```bash
#!/bin/bash
TEMPLATE='{{$i:=false}}{{$r:=false}}{{$w:=false}}{{$s:=false}}{{range .Sessions}}{{range .Signals}}{{if eq .State "idle"}}{{$i = true}}{{end}}{{if eq .State "running"}}{{$r = true}}{{end}}{{if eq .State "waiting"}}{{$w = true}}{{end}}{{if eq .State "started"}}{{$s = true}}{{end}}{{end}}{{end}}{{if $i}}#[fg=cyan]⏺#[default]{{else}}#[fg=white]⏺#[default]{{end}}{{if $r}}#[fg=green]⏺#[default]{{else}}#[fg=white]⏺#[default]{{end}}{{if $w}}#[fg=yellow]⏺#[default]{{else}}#[fg=white]⏺#[default]{{end}}{{if $s}}#[fg=blue]⏺#[default]{{else}}#[fg=white]⏺#[default]{{end}}'
beacon scan --scope session -a --template "$TEMPLATE" 2>/dev/null
```

States: idle (cyan), running (green), waiting (yellow), started (blue)

Add to your `tmux.conf`:

```
set -g status-right '#(beacon-status)'
```

## License

MIT

## Author

[monochromegane](https://github.com/monochromegane)
