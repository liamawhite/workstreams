#!/bin/bash

# Copyright 2025 Liam White
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Shell wrapper for workstreams CLI that handles directory changes.
# Usage: source this file or add the workstreams function to your shell profile.

workstreams() {
    local cmd="$1"

    # Commands that may change the working directory (includes empty cmd for root TUI)
    if [[ -z "$cmd" || "$cmd" == "add" || "$cmd" == "switch" || "$cmd" == "sw" || "$cmd" == "remove" || "$cmd" == "rm" ]]; then
        local temp_file
        temp_file=$(mktemp)

        # Run command with stderr redirected to temp file while still displaying it
        command workstreams "$@" 2> >(tee "$temp_file" >&2)
        local exit_code=$?

        # Look for directory change or exit indicators in stderr
        local target_dir should_exit
        target_dir=$(grep "^WS_CHDIR:" "$temp_file" 2>/dev/null | sed 's/^WS_CHDIR://')
        should_exit=$(grep "^WS_EXIT$" "$temp_file" 2>/dev/null)

        rm -f "$temp_file"

        if [[ -n "$target_dir" && -d "$target_dir" ]]; then
            cd "$target_dir" || echo "Warning: Failed to change to directory: $target_dir"
        fi

        if [[ -n "$should_exit" ]]; then
            exit $exit_code
        fi

        return $exit_code
    else
        command workstreams "$@"
    fi
}

# Completion support (if the binary supports it)
if command -v workstreams >/dev/null 2>&1; then
    complete -F _workstreams workstreams 2>/dev/null || true
fi
