# workstreams

A lightweight CLI for switching between project workstreams. Run `workstreams` to pick where you want to be; the shell lands you there automatically.

## Installation

Download the latest binary for your platform from the [releases page](https://github.com/liamawhite/workstreams/releases), then put it somewhere on your `$PATH`.

Or install from source:

```sh
go install github.com/liamawhite/workstreams@latest
```

### Shell integration

The CLI needs a shell wrapper to change your working directory. Add this to your `.bashrc` or `.zshrc`:

```sh
source /path/to/workstreams.sh
```

`workstreams.sh` is included in the release archive and in the repo root.

## Getting started

Every workstream is a directory under `~/workstreams/<name>`. Creating one drops you straight into it:

```sh
workstreams add "my project"
# Created workstream "my project"
# → now in ~/workstreams/my-project
```

To jump between workstreams, run `workstreams` with no arguments to open the interactive picker, or switch by name directly:

```sh
workstreams switch my-project   # or: workstreams sw my-project
# → now in ~/workstreams/my-project
```

When you're done with a workstream:

```sh
workstreams remove my-project   # or: workstreams rm my-project
```

If you were inside the removed workstream, you're moved back to `~/workstreams` automatically.

## Types

Types are reusable templates that run setup and load hooks whenever you create or switch to a workstream of that type. Useful for cloning repos, activating virtual environments, setting up tmux sessions, etc.

### Creating a type

```sh
workstreams types add go-service
# Created type "go-service" at ~/.workstreams/types/go-service
```

This creates a directory at `~/.workstreams/types/go-service/` where you add hook scripts.

### Hooks

| File | When it runs | Behaviour |
|------|-------------|-----------|
| `onInit.sh` | Once, when the workstream is created | Runs synchronously — output streams to the terminal |
| `onInitAsync.sh` | Once, after `onInit.sh` completes | Runs detached — output goes to `.async-init.log` in the workstream dir |
| `onLoad.sh` | Every time you switch to the workstream | Runs synchronously before the directory change |

All hooks receive these environment variables:

```
WORKSTREAM_DIR   — absolute path to the workstream directory
WORKSTREAM_NAME  — display name (e.g. "my project")
WORKSTREAM_TYPE  — type name (e.g. "go-service")
```

### Example: a `go-service` type

**`~/.workstreams/types/go-service/onInit.sh`**
```sh
#!/usr/bin/env bash
gh repo clone "my-org/${WORKSTREAM_NAME}" "$WORKSTREAM_DIR/repo"
```

**`~/.workstreams/types/go-service/onLoad.sh`**
```sh
#!/usr/bin/env bash
cd "$WORKSTREAM_DIR/repo"
```

Then create workstreams from that type:

```sh
workstreams add "payments service" --type go-service
```

## Scripting and shell prompts

`workstreams current` prints the display name of the workstream you're currently in:

```sh
workstreams current
# my project
```

Returns an error if you're not inside any workstream. Handy for shell prompts or scripts that need to know context:

```sh
# zsh prompt snippet
PS1='$(workstreams current 2>/dev/null) $ '
```

## Directory layout

```
~/workstreams/
  my-project/
    config.yaml       ← name and type for this workstream
    ...               ← your files

~/.workstreams/
  types/
    go-service/
      onInit.sh
      onInitAsync.sh
      onLoad.sh
```
