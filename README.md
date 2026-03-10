# Marvin Time Tracker

A minimal iOS app + Go relay server that surfaces a live timer via Live Activity whenever [Amazing Marvin](https://amazingmarvin.com) time tracking is active.

The Go server bridges Marvin webhooks to Apple's ActivityKit push notifications, enabling real-time timer display on iPhone Lock Screen, Dynamic Island, and Apple Watch Smart Stack.

## Architecture

```
Marvin Client (web/desktop/mobile)
    |
    | webhook (startTracking / stopTracking)
    v
Go Relay Server ──> APNs ──> iPhone Live Activity
                                   + Apple Watch Smart Stack
```

## Components

- **`server/`** — Go relay server (single binary, ~6-8MB)
- **`ios/`** — SwiftUI iOS app (iOS 18+, XcodeGen)

## Quick Start

All commands use the [Justfile](https://github.com/casey/just) (`just --list` to see all recipes).

### Server

```bash
cp server/config.example server/config
# Edit server/config with your tokens and keys
just build
just run
```

### iOS App

```bash
cd ios && bundle install && cd ..
just ios-deploy    # Build, install, launch on device
```

## Releasing a New Version

Pushing a tag triggers the full release pipeline — Homebrew formula, server binary, and userscript are all updated automatically.

### 1. Bump and push

```bash
just release --dry-run   # Preview next version
just release             # Bump, changelog, tag, and push
```

Cocogitto determines the version from conventional commits (`feat:` → minor, `fix:` → patch, `feat!:` → major). The `bump-homebrew.yml` workflow runs automatically on tag push, updating the formula in the [Homebrew tap](https://github.com/strubio-ray/homebrew-tap).

### 2. Update the server

On the machine running the relay server:

```bash
brew update
brew upgrade marvin-relay
brew services restart marvin-relay
```

The new binary includes the updated userscript via `go:embed`.

### 3. Userscript updates

Once the server restarts, it serves the latest userscript at `/userscript/marvin-relay-tracker.user.js`. The userscript is embedded in the binary via `go:embed`, so every server release serves the latest version automatically.

However, **Tampermonkey only detects an update when `@version` increases**. If a release only changes server-side code and not the userscript, Tampermonkey correctly sees no update. When the userscript itself changes, bump `@version` in `userscript/marvin-relay-tracker.user.js` before tagging — otherwise Tampermonkey will offer a reinstall instead of an update.

Auto-update requires `EXTERNAL_URL` to be set in the server config. See [userscript/README.md](userscript/README.md) for more details on update methods and first-time install.

### Prerequisites

- `HOMEBREW_TAP_TOKEN` secret in this repo — a fine-grained PAT scoped to the tap repo with `Contents: Read and write`
- `EXTERNAL_URL` in the server config — for userscript auto-update URL rewriting
- Bump `@version` in `userscript/marvin-relay-tracker.user.js` when the userscript changes

## Requirements

- Go 1.22+
- iOS 18+ / watchOS 11+
- Apple Developer account (for APNs p8 key)
- Amazing Marvin account with API token
- Tailscale Funnel (or any HTTPS reverse proxy) for webhook delivery
