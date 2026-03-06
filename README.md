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
    |                              + Apple Watch Smart Stack
    | adaptive polling fallback
    v
Marvin API (GET /api/trackedItem)
```

## Components

- **`server/`** — Go relay server (single binary, ~6-8MB)
- **`ios/`** — SwiftUI iOS app (iOS 18+, XcodeGen)

## Quick Start

### Server

```bash
cp .env.example .env
# Edit .env with your tokens and keys
make build
make run
```

### iOS App

```bash
cd ios
bundle install
bundle exec fastlane setup
open MarvinTimeTracker.xcodeproj
```

## Requirements

- Go 1.22+
- iOS 18+ / watchOS 11+
- Apple Developer account (for APNs p8 key)
- Amazing Marvin account with API token
- Tailscale Funnel (or any HTTPS reverse proxy) for webhook delivery
