# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Marvin Time Tracker: a minimal iOS app + Go relay server that surfaces a live timer via Live Activity whenever [Amazing Marvin](https://amazingmarvin.com) time tracking is active. The Go server bridges Marvin webhooks to Apple's ActivityKit push notifications.

## Build & Test Commands

All commands use the Justfile (`just --list` to see all recipes).

### Go Server
```bash
just build          # Build server binary to server/marvin-relay
just test           # Run all server tests
just run            # Build and run server
just clean          # Remove built binary
just deploy-dev     # Test + build + install to Homebrew + restart (development)
just deploy-prod    # Same as above but with APNS_ENV=production

# Run a single test (no just recipe for this)
go test ./server/... -run TestFunctionName
```

### iOS App
```bash
just ios-deploy      # Build, install, launch on device via Fastlane
just ios-testflight  # Build and upload to TestFlight
```

For Fastlane-only operations that don't have Justfile recipes:
```bash
cd ios
bundle exec fastlane setup           # Generate project + sync dev signing
bundle exec fastlane sync_certs      # Sync certificates via match
bundle exec fastlane build           # Release build only (no upload)
```

The iOS project uses **XcodeGen** (`ios/project.yml`) — there is no checked-in `.xcodeproj`. Regenerate after changing targets, sources, or settings.

Version is managed in `ios/version.xcconfig` (`MARKETING_VERSION` and `CURRENT_PROJECT_VERSION`). TestFlight builds auto-increment the build number from the latest TestFlight build.

## Architecture

### Go Server (`server/`)

Single-binary relay server using Go 1.22+ stdlib `net/http` routing. Two external deps: `sideshow/apns2` (APNs push) and `rs/cors`.

Key files and their roles:
- **`main.go`** — Wires config, state store, dedup, APNs client, renewal, and server
- **`server.go`** — HTTP mux setup, CORS config, status endpoint. Uses functional options (`ServerOption`)
- **`webhook.go`** — Handles `POST /webhook/start` and `/webhook/stop` from Marvin client-side AJAX
- **`register.go`** — `POST /register` receives push tokens from the iOS app
- **`track.go`** — `POST /start` and `/stop` for app-initiated tracking via Marvin API
- **`state.go`** — `StateStore` with JSON file persistence and atomic rename. Holds tracking state + push tokens
- **`dedup.go`** — Deduplicates Marvin's duplicate webhook firings (~9s apart) using composite key
- **`renewal.go`** — Handles 8-hour Live Activity cap by ending and restarting activities at ~7h45m
- **`apns.go`** — APNs client wrapper using `sideshow/apns2` with JWT auth
- **`notifier.go`** — `Notifier` interface abstracting push notification delivery (enables test mocks)
- **`config.go`** — Environment variable loading with defaults
- **`marvin.go`** — Marvin API client (`MarvinAPIClient` interface)
- **`auth.go`** — `requireAPIKey` middleware for Bearer token auth on app-facing endpoints
- **`tasks.go`** — `GET /tasks` handler proxying Marvin's `/todayItems` endpoint

State machine: `IDLE <-> TRACKING`, persisted to JSON file. Webhooks drive state transitions.

### iOS App (`ios/`)

SwiftUI app targeting iOS 18+ / watchOS 11+. Swift 6.0. Uses `@Observable` (no TCA/coordinators).

- **`MarvinTimeTracker/`** — Main app target
  - `Views/` — OnboardingView (API key entry), TimerView (main screen), TaskPickerSheet
  - `ViewModels/TrackingViewModel.swift` — `@Observable`, manages API calls + Live Activity lifecycle
  - `Services/` — MarvinAPIClient, KeychainService (native Security framework), PushTokenService
  - `Models/` — TrackingState, MarvinTask
- **`MarvinTimeTrackerWidgets/`** — Widget extension for Live Activity UI (Lock Screen, Dynamic Island, Watch Smart Stack)
- **`Shared/TimeTrackerAttributes.swift`** — `ActivityAttributes` shared between app and widget extension

Watch support is auto-mirrored Live Activities via `.supplementalActivityFamilies([.small])` — no watchOS target.

### Data Flow

```
Marvin Client → webhook → Go Server → APNs → iPhone Live Activity / Watch Smart Stack
iOS App → POST /register → Go Server (stores push tokens)
iOS App → POST /start|/stop → Marvin API (via Go server proxy)
```

## Configuration

Server configured via config file and/or env vars (see `server/config.example`):
- `MARVIN_API_TOKEN` (required)
- `MARVIN_FULL_ACCESS_TOKEN` (required)
- `API_KEY` (optional, but strongly recommended — protects app-facing endpoints)
- `APNS_KEY_ID`, `APNS_TEAM_ID`, `APNS_KEY_P8_PATH`, `APNS_BUNDLE_ID`
- `STATE_FILE_PATH`, `LISTEN_ADDR`

The iOS app authenticates with the server using `API_KEY` via `Authorization: Bearer` header. The Marvin API tokens never leave the server.

iOS signing requires:
- `DEVELOPMENT_TEAM` — Apple Developer Team ID (used in `project.yml`)
- `ASC_KEY_ID`, `ASC_ISSUER_ID`, `ASC_KEY_P8_PATH` — App Store Connect API key (for Fastlane match and TestFlight)

## Key Design Decisions

- CORS must return status `200` on OPTIONS (not 204) — Marvin requires this
- Webhooks are client-side AJAX from the Marvin web/desktop app
- Live Activities have an 8-hour system cap; server auto-renews at 7h45m
- APNs `liveactivity` push type requires p8 key (not p12)
- `Notifier` interface in `notifier.go` enables testing without real APNs
- Fastlane sets `SKIP_COG=1` to bypass cocogitto commit hooks for its auto-generated commits
- Code signing uses Fastlane Match (manual style) — profiles are referenced by name in `project.yml`
- Bundle ID: `com.strubio.MarvinTimeTracker`
- iOS app uses `marvin-tracker://` URL scheme for deep links (e.g., Stop button in Live Activity)
- App-facing endpoints (`/status`, `/register`, `/start`, `/stop`, `/tasks`) require `Authorization: Bearer <API_KEY>` when `API_KEY` is configured
- Webhook and public endpoints (`/webhook/*`, `/events`, `/history`, `/userscript/*`) are unauthenticated

## Release Pipeline

**Every server or iOS change requires a new git tag to reach Homebrew installations.** See [README.md § Releasing a New Version](README.md#releasing-a-new-version) for the full workflow.

Quick reference:
```bash
git tag -l 'v*' --sort=-v:refname | head -5   # Check existing tags
git tag v<next>                                 # Create new tag
git push origin v<next>                         # Triggers bump-homebrew.yml
```

Then on the deployment machine: `brew update && brew upgrade marvin-relay && brew services restart marvin-relay`

**If you skip the tag, the Homebrew formula won't update and the server won't pick up new changes.** The `bump-homebrew.yml` GitHub Action only fires on tag push (`v*`).
