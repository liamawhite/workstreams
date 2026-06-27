package types

import "path/filepath"

var onInitBoilerplate = []byte(`#!/bin/bash
# onInit.sh — runs synchronously when a workstream is created; output is streamed
# to the terminal. The workstreams command does not return until this completes.
# Environment variables: WORKSTREAM_DIR, WORKSTREAM_NAME, WORKSTREAM_TYPE

touch "$WORKSTREAM_DIR/AGENTS.md"
printf '@AGENTS.md\n' > "$WORKSTREAM_DIR/CLAUDE.md"
`)

var onInitAsyncBoilerplate = []byte(`#!/bin/bash
# onInitAsync.sh — runs in the background after onInit.sh completes.
# The workstreams command returns immediately; this script outlives it.
# stdout/stderr are written to .async-init.log inside the workstream directory.
# Environment variables: WORKSTREAM_DIR, WORKSTREAM_NAME, WORKSTREAM_TYPE
`)

var onLoadBoilerplate = []byte(`#!/bin/bash
# onLoad.sh — runs whenever you switch to a workstream of this type.
# Environment variables: WORKSTREAM_DIR, WORKSTREAM_NAME, WORKSTREAM_TYPE
`)

// OnInitScript returns the path to the onInit.sh hook for the given type.
func OnInitScript(dirName string) string {
	dir, _ := TypeDir(dirName)
	return filepath.Join(dir, "onInit.sh")
}

// OnInitAsyncScript returns the path to the onInitAsync.sh hook for the given type.
func OnInitAsyncScript(dirName string) string {
	dir, _ := TypeDir(dirName)
	return filepath.Join(dir, "onInitAsync.sh")
}

// OnLoadScript returns the path to the onLoad.sh hook for the given type.
func OnLoadScript(dirName string) string {
	dir, _ := TypeDir(dirName)
	return filepath.Join(dir, "onLoad.sh")
}
