# Project Instructions

## After Every Go Code Change

After making Go code changes, you MUST run the following commands in order and fix any issues:

1. `bwapp dev gen` — regenerate generated code
2. `bwapp dev fmt` — format all files
3. `bwapp check lint` — lint and fix all errors
4. `bwapp check unit-test` — run unit tests and fix any failures