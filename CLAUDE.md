# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Marvin Time Tracker: a minimal iOS app + Go relay server that surfaces a live timer via Live Activity whenever [Amazing Marvin](https://amazingmarvin.com) time tracking is active. The Go server bridges Marvin webhooks to Apple's ActivityKit push notifications.

## Build & Test Commands

### Go Server
```bash
make build          # Build server binary to server/marvin-relay
make test           # Run all server tests
make run            # Build and run server
make clean          # Remove built binary

# Run a single test
go test ./server/... -run TestFunctionName

# Cross-compile for deployment
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o server/marvin-relay ./server
```

### iOS App
```bash
cd ios
xcodegen generate                    # Generate .xcodeproj from project.yml
open MarvinTimeTracker.xcodeproj     # Open in Xcode

# Fastlane (run from ios/ directory)
bundle exec fastlane setup           # Generate project + sync dev signing
bundle exec fastlane deploy          # Build, install, launch on device
bundle exec fastlane sync_certs      # Sync certificates via match
bundle exec fastlane build           # Release build only (no upload)
bundle exec fastlane testflight_release  # Build + upload to TestFlight
```

The iOS project uses **XcodeGen** (`ios/project.yml`) — there is no checked-in `.xcodeproj`. Regenerate after changing targets, sources, or settings.

Version is managed in `ios/version.xcconfig` (`MARKETING_VERSION` and `CURRENT_PROJECT_VERSION`). TestFlight builds auto-increment the build number from the latest TestFlight build.

## Architecture

### Go Server (`server/`)

Single-binary relay server using Go 1.22+ stdlib `net/http` routing. Two external deps: `sideshow/apns2` (APNs push) and `rs/cors`.

Key files and their roles:
- **`main.go`** — Wires config, state store, dedup, APNs client, poller, renewal, and server
- **`server.go`** — HTTP mux setup, CORS config, status endpoint. Uses functional options (`ServerOption`)
- **`webhook.go`** — Handles `POST /webhook/start` and `/webhook/stop` from Marvin client-side AJAX
- **`register.go`** — `POST /register` receives push tokens from the iOS app
- **`track.go`** — `POST /start` and `/stop` for app-initiated tracking via Marvin API
- **`state.go`** — `StateStore` with JSON file persistence and atomic rename. Holds tracking state + push tokens
- **`dedup.go`** — Deduplicates Marvin's duplicate webhook firings (~9s apart) using composite key
- **`poller.go`** — Adaptive fallback polling of Marvin API (`/api/trackedItem`); adjusts interval based on tracking state
- **`quota.go`** — Daily API call budget manager (1,440/day Marvin limit)
- **`renewal.go`** — Handles 8-hour Live Activity cap by ending and restarting activities at ~7h45m
- **`apns.go`** — APNs client wrapper using `sideshow/apns2` with JWT auth
- **`notifier.go`** — `Notifier` interface abstracting push notification delivery (enables test mocks)
- **`config.go`** — Environment variable loading with defaults
- **`marvin.go`** — Marvin API client (`MarvinAPIClient` interface)

State machine: `IDLE <-> TRACKING`, persisted to JSON file. Webhooks are primary; polling is fallback.

### iOS App (`ios/`)

SwiftUI app targeting iOS 18+ / watchOS 11+. Swift 6.0. Uses `@Observable` (no TCA/coordinators).

- **`MarvinTimeTracker/`** — Main app target
  - `Views/` — OnboardingView (token entry), TimerView (main screen), TaskPickerSheet
  - `ViewModels/TrackingViewModel.swift` — `@Observable`, manages API calls + Live Activity lifecycle
  - `Services/` — MarvinAPIClient, KeychainService (native Security framework), PushTokenService
  - `Models/` — TrackingState, MarvinTask
- **`MarvinTimeTrackerWidgets/`** — Widget extension for Live Activity UI (Lock Screen, Dynamic Island, Watch Smart Stack)
- **`Shared/TimeTrackerAttributes.swift`** — `ActivityAttributes` shared between app and widget extension

Watch support is auto-mirrored Live Activities via `.supplementalActivityFamilies([.small])` — no watchOS target.

### Data Flow

```
Marvin Client → webhook → Go Server → APNs → iPhone Live Activity / Watch Smart Stack
                              ↑
                    fallback polling of Marvin API
iOS App → POST /register → Go Server (stores push tokens)
iOS App → POST /start|/stop → Marvin API (via Go server proxy)
```

## Configuration

Server configured via env vars (see `.env.example` in project root):
- `MARVIN_API_TOKEN` (required)
- `APNS_KEY_ID`, `APNS_TEAM_ID`, `APNS_KEY_P8_PATH`, `APNS_BUNDLE_ID`
- `STATE_FILE_PATH`, `LISTEN_ADDR`, `POLL_INTERVAL_ACTIVE`, `POLL_INTERVAL_IDLE`

iOS signing requires:
- `DEVELOPMENT_TEAM` — Apple Developer Team ID (used in `project.yml`)
- `ASC_KEY_ID`, `ASC_ISSUER_ID`, `ASC_KEY_P8_PATH` — App Store Connect API key (for Fastlane match and TestFlight)

## Key Design Decisions

- CORS must return status `200` on OPTIONS (not 204) — Marvin requires this
- Webhooks are client-side AJAX, so delivery is unreliable; polling provides redundancy
- Live Activities have an 8-hour system cap; server auto-renews at 7h45m
- APNs `liveactivity` push type requires p8 key (not p12)
- `Notifier` interface in `notifier.go` enables testing without real APNs
- Fastlane sets `SKIP_COG=1` to bypass cocogitto commit hooks for its auto-generated commits
- Code signing uses Fastlane Match (manual style) — profiles are referenced by name in `project.yml`
- Bundle ID: `com.strubio.MarvinTimeTracker`
