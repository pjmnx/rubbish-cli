# rubbish-cli

[![Release](https://img.shields.io/github/v/release/pjmnx/rubbish-cli?sort=semver)](https://github.com/pjmnx/rubbish-cli/releases)
![Release Date](https://img.shields.io/github/release-date/pjmnx/rubbish-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
![Go Version](https://img.shields.io/badge/go-1.20%2B-blue)
[![Go Report Card](https://goreportcard.com/badge/github.com/pjmnx/rubbish-cli)](https://goreportcard.com/report/github.com/pjmnx/rubbish-cli)

Trash management for the terminal. Rubbish is a safe alternative to `rm`: instead of permanently deleting files, it “tosses” them into a per-user container where they remain for a retention period. You can list, inspect, restore, and eventually wipe them when they expire.

## Features

- Toss files/directories to a container instead of deleting.
- Default retention is 30 days; override per run.
- List status locally (per working directory) or globally.
- Inspect item details (origin, tossed at, wipeable at, remaining time).
- Restore items back to the working directory.
- Wipe items (locally or globally), with optional force/auto-confirm.
- Persistent journal using BadgerDB.
- Per user cofirguration wover

## Install

Prereqs: Go 1.20+ (module targets newer Go in `go.mod`).

- Prebuilt binaries (recommended)

	Download the latest release for your OS/arch from the Releases page, then install:

```bash
# 1) Download from https://github.com/pjmnx/rubbish-cli/releases
# 2) Make executable and move into PATH
chmod +x rubbish_<os>_<arch>
sudo mv rubbish_<os>_<arch> /usr/local/bin/rubbish
rubbish --version
```

- From source

```bash
git clone https://github.com/pjmnx/rubbish-cli.git
cd rubbish-cli
go build -o bin/rubbish .
```

Optionally put it on PATH:

```bash
sudo mv bin/rubbish /usr/local/bin/
```

## Configuration

Rubbish loads two config files (first is system, second is user):

- `/etc/rubbish/config.cfg`
- `~/.config/rubbish.cfg`

Settings include:

- `wipeout_time` (int, days) – default retention, e.g. `30`
- `container_path` (string) – where tossed files are stored; `~` expands
- `max_retention`, `cleanup_interval`
- `[notifications] enabled, days_in_advance, timeout`

Example user config `~/.config/rubbish.cfg`:

```ini
[DEFAULT]
wipeout_time = 30
container_path = ~/.local/share/rubbish
[notifications]
enabled = false
days_in_advance = 3
timeout = 5
```

On first run, the tool will create the container directory if it does not exist and open a journal at `<container_path>/.journal`.

## Usage

General:

```bash
rubbish <command> [options] [args]
```

Show help:

```bash
rubbish help
rubbish help <command>
```

### Commands

- toss – Move files/dirs to the container
	- Flags: `-r <days>` retention override, `-s` silent
	- Example:
		```bash
		rubbish toss -r=7 my.log docs/
		```

- status – Show items; local by default, `-g` for global
	- Example:
		```bash
		rubbish status        # only items from current working dir subtree
		rubbish status -g     # all items
		```

- info – Show details for an item or by position
	- Flags: `-p <n>` 1-based position; negative selects from the end
	- Examples:
		```bash
		rubbish info file.txt
		rubbish info -p=1     # first item
		rubbish info -p=-1    # last item
		```

- restore – Restore items into the current directory
	- Flags: `--override` (or `-o` if you wire it) to overwrite existing files, `--silent`/`-s`
	- Example:
		```bash
		rubbish restore file.txt other.doc
		```

- wipe – Permanently remove items
	- Flags: `-f` ignore retention (force), `-y` auto-confirm, `-g` global
	- Examples:
		```bash
		rubbish wipe          # local wipe of wipeable items (asks per item)
		rubbish wipe -g -y    # wipe all wipeable items globally without prompt
		rubbish wipe -f file1 file2   # force wipe specific items
		```

## How it works

- Tossing moves the file to `<container_path>/<basename>_<RANDOM>` and records metadata in the journal (origin path, tossed time, retention days).
- Status filters by current working directory when not `-g`.
- Info formats toss time, wipeable date, and remaining/overdue time.
- Wipe removes the file from the container and deletes the journal record (with confirmation unless `-y`).
- Restore moves the file back to the current directory; use `--override` to replace existing files.

## Notes

- The journal backend is BadgerDB stored at `<container_path>/.journal`.
- `container_path` may use `~` and is normalized to an absolute path.
- Bin size is computed excluding the `.journal` directory.

## Development

Run tests:

```bash
go test ./...
```

Helpful samples are in `tests/`.

## Troubleshooting

- “Unknown command” – run `rubbish help` to see available commands.
- “Container path does not exist” – Rubbish will try to create it; ensure you have permissions.
- Journal errors – verify `<container_path>/.journal` is writable.


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.