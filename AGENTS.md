# What is this?

YggLazy is an open-source multiplatform tool for easily configuring the [Yggdrasil Network](https://yggdrasil-network.github.io/) without hassle, written in Go.

> YggLazy helps you quickly bring up the Yggdrasil Network on any machine without manually copying peer addresses. It pings peers by region, ranks the best ones, and lets you choose how many of those top peers will be added to `yggdrasil.conf`.

# Repo Structure

- All core code is in the root directory: `main.go` and platform-specific files (`platform_*.go`).
- `docs/` — documentation.
- `install.sh` / `uninstall.sh` — multi-distribution Unix install/uninstall scripts.
- `linux-install/` — desktop integration files for Linux/Unix (freedesktop standards).
- `.goreleaser.yaml` — CI release configuration.
- `yggdrasil.conf.schema` — example Yggdrasil config; contains no private or public keys.

# Rules

- Keep it simple.
- Do not edit code in response to a question.
- Never commit, push, or open a PR unless asked.
- If the task is not fully clear or can be solved in several ways, ask the user for the preferred approach.

# Tests

- A test must fail on broken code. Otherwise, do not write it.

# Commit Messages

> area: short lowercase explanation

- For example:
  `scripts: install|uninstall now have full unix support (BSDs, other certified unix)`

Description only if required, in simple language.