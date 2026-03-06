# Marvin Time Tracker — Design Document

## Overview

A minimal iOS app (SwiftUI, iOS 18+) that surfaces a live timer via Live Activity whenever Amazing Marvin time tracking is active — regardless of which Marvin client started it. A Go relay server bridges Marvin webhooks to Apple's ActivityKit push notifications.

### Design Principles

- **Live Activity is the product** — the app itself is a thin launcher
- **Timer-only V1** — no task management, no browsing, no inbox
- **Portable server** — single Go binary, deploy anywhere
- **Reliability through redundancy** — webhooks first, adaptive polling as fallback

### Targets

- **iOS app:** iOS 18+ / watchOS 11+ (SwiftUI)
- **Server:** Go 1.22+ single binary
- **Distribution:** Sideloaded via Xcode (personal use)
- **Watch:** Auto-mirrored Live Activity in Smart Stack (zero watchOS code)

---

## Architecture

```
Marvin Client (web/desktop/mobile)
    |
    | webhook (startTracking / stopTracking)
    v
Go Relay Server (single binary)
    |
    |-- Dedup + authoritative state machine (JSON file)
    |-- Adaptive fallback polling (time.Ticker)
    |-- sideshow/apns2 (HTTP/2 + JWT)
    |
    | (on state change)
    v
APNs (ActivityKit push)
    |
    |---> iPhone: Live Activity
    |       - Lock Screen timer
    |       - Dynamic Island timer
    |       - Interactive stop button (LiveActivityIntent)
    |
    '---> Apple Watch: Smart Stack (auto-mirrored)

iOS App (SwiftUI)
    |-- Onboarding (token + notification permissions)
    |-- Timer screen (current task + elapsed time + stop)
    |-- Start tracking sheet (today's tasks)
    '-- Sends push tokens to Go server
```

---

## Go Relay Server

### Dependencies

| Dependency | Purpose |
|---|---|
| `sideshow/apns2` | APNs HTTP/2 + JWT client (4.6K stars, maintained) |
| `rs/cors` | CORS middleware for Marvin webhook preflight |
| Go stdlib `net/http` | HTTP server and routing (Go 1.22+ enhanced routing) |

No framework needed. Two external dependencies.

### Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/webhook/start` | Receives Marvin `startTracking` webhook |
| `POST` | `/webhook/stop` | Receives Marvin `stopTracking` webhook |
| `OPTIONS` | `/webhook/*` | CORS preflight (must return 200, not 204) |
| `POST` | `/register` | iOS app sends push tokens (start + update) |
| `GET` | `/status` | Health check + current tracking state |

### CORS Configuration

Marvin webhooks are client-side AJAX. The server must return:
- `Access-Control-Allow-Origin: *` (wildcard for web + desktop + mobile clients)
- `Access-Control-Allow-Methods: GET, POST, PUT, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type`
- Status `200` on OPTIONS (not 204 — Marvin requires exactly 200)
- HTTPS only (HTTP endpoints are silently ignored by Marvin)

### State Machine

```
         webhook:start          webhook:stop
  IDLE -----------------> TRACKING -----------------> IDLE
   ^                         |                          |
   |                         | poll confirms stopped    |
   '-------------------------'                          |
   |                                                    |
   '----------------------------------------------------'
              poll confirms no tracked item
```

State stored as a JSON file with atomic rename:

```json
{
  "trackingTaskId": "C3Z5BdjaX7wn5mYa9svk",
  "taskTitle": "Really figure out what's wrong with my back",
  "startedAt": 1772734813781,
  "pushToStartToken": "<hex>",
  "updateToken": "<hex>",
  "liveActivityStartedAt": "2026-03-05T10:00:13Z",
  "lastPollAt": "2026-03-05T10:05:00Z",
  "lastWebhookAt": "2026-03-05T10:00:13Z"
}
```

### Webhook Processing

1. Return `200 OK` immediately (acknowledge-first)
2. Parse task payload from request body
3. Determine event type from URL path (`/webhook/start` vs `/webhook/stop`)
4. Dedup: hash `taskId + round(timestamp, 15s)` — collapses Marvin's ~9-second duplicate firings
5. Compare against current state machine
6. If state changed: send ActivityKit push via APNs
7. If start: push-to-start or update Live Activity with `startedAt` timestamp
8. If stop: end Live Activity with final elapsed time

### Adaptive Fallback Polling

Polls `GET /api/trackedItem` from the Marvin API. Adapts frequency based on context:

| Context | Poll Interval | Daily Budget Impact |
|---|---|---|
| Active tracking (timer running) | Every 30-60 seconds | ~48-120 calls/active hour |
| Idle (no tracking) | Every 5-10 minutes | ~144-288 calls/day |
| Quiet hours (configurable) | Every 30 minutes | ~16 calls/night |
| After webhook received | Skip next scheduled poll | Saves ~1 call/event |

A quota-aware budget manager tracks calls consumed against the 1,440/day limit. Reserves 5% for emergencies. Distributes remaining calls over remaining hours to prevent morning exhaustion.

### APNs Integration

Uses `sideshow/apns2` with JWT (token-based) authentication:

- **Auth:** p8 key file + Key ID + Team ID (p12 not supported for `liveactivity` push type)
- **Push type:** `liveactivity`
- **Topic:** `{bundleId}.push-type.liveactivity`
- **Push-to-start:** Sends with `apns-push-type: liveactivity` to create a new Live Activity
- **Update:** Sends with the activity's update token to modify `ContentState`
- **End:** Sends `"event": "end"` with final content and `"dismissal-date"` for cleanup

### 8-Hour Live Activity Renewal

Live Activities have an 8-hour system cap. The server handles this transparently:

1. Server tracks `liveActivityStartedAt` in state
2. At ~7h45m elapsed, server ends the current Live Activity (`"event": "end"`, `"dismissal-date": <now>`)
3. Server immediately sends push-to-start to create a new Live Activity
4. New `ContentState` carries the **original** `startedAt` timestamp
5. `Text(timerInterval:)` continues counting from the original start — seamless to the user
6. Brief visual flicker is unavoidable (no atomic swap API)

### Configuration

All via environment variables:

| Variable | Description |
|---|---|
| `APNS_KEY_ID` | Apple p8 key identifier |
| `APNS_TEAM_ID` | Apple Developer Team ID |
| `APNS_KEY_P8_PATH` | Path to `.p8` key file |
| `APNS_BUNDLE_ID` | iOS app bundle identifier |
| `MARVIN_API_TOKEN` | Marvin API token for fallback polling |
| `STATE_FILE_PATH` | Path to JSON state file (default: `./state.json`) |
| `LISTEN_ADDR` | Server listen address (default: `:8080`) |
| `POLL_INTERVAL_ACTIVE` | Polling interval during tracking (default: `30s`) |
| `POLL_INTERVAL_IDLE` | Polling interval when idle (default: `5m`) |

### Deployment Options

| Option | Cost | Notes |
|---|---|---|
| Oracle Cloud free ARM VPS | Free | 4 cores, 24GB RAM, genuinely free forever. Use Caddy for auto-TLS. |
| Fly.io | ~$1.94/mo | Managed, easy deploys, scale-to-zero available |
| Any VPS + systemd | Varies | `scp` binary + systemd unit file |
| Docker | N/A | `FROM scratch` image, <20MB |
| Self-hosted (Coolify, etc.) | Varies | Docker-based, auto-TLS |

Cross-compilation: `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o marvin-relay`

### Binary Size

~6-8MB static binary. No runtime dependencies. No container runtime needed for bare metal deployment.

---

## iOS App

### Minimum Deployment Target

- iOS 18.0+ (for Watch Smart Stack mirroring, reliable push-to-start tokens)
- watchOS 11.0+ (auto-mirrored Live Activities)

### Architecture

Plain SwiftUI with `@Observable`. No TCA, no coordinator, no DI container.

```
MarvinTimeTracker/
|-- MarvinTimeTrackerApp.swift        // @main, root view switching
|-- Models/
|   |-- TrackingState.swift           // Current tracking state model
|   '-- MarvinTask.swift              // Task model from API
|-- ViewModels/
|   '-- TrackingViewModel.swift       // @Observable, API + Live Activity lifecycle
|-- Views/
|   |-- OnboardingView.swift          // Token entry + permission request
|   |-- TimerView.swift               // Current task + elapsed time + stop
|   '-- TaskPickerSheet.swift         // Today's tasks list for starting tracking
|-- Services/
|   |-- MarvinAPIClient.swift         // API calls (track, todayItems, trackedItem)
|   |-- KeychainService.swift         // Secure token storage
|   '-- PushTokenService.swift        // Token registration with Go server
|-- LiveActivity/
|   |-- TimeTrackerAttributes.swift   // ActivityAttributes + ContentState
|   '-- TimeTrackerLiveActivity.swift // ActivityConfiguration + Watch layout
'-- Info.plist
    |-- NSSupportsLiveActivities: YES
    '-- NSSupportsLiveActivitiesFrequentUpdates: YES
```

~10 files. No navigation router needed for two screens.

### Screens

**1. Onboarding (shown when no token in Keychain)**
- `SecureField` for pasting Marvin API token
- Validate with `GET /api/me`
- Request notification permission
- Register `pushToStartTokenUpdates` and send to Go server

**2. Timer View (main screen)**
- If tracking: task title + `Text(timerInterval:)` elapsed timer + Stop button
- If idle: "No task being tracked" + "Start Tracking" button
- "Start Tracking" presents `TaskPickerSheet`

**3. Task Picker Sheet**
- Flat list from `GET /api/todayItems`
- Tap to call `POST /api/track` with `{taskId, action: "START"}`
- Dismisses on selection

### Live Activity

**ActivityAttributes:**

```swift
struct TimeTrackerAttributes: ActivityAttributes {
    struct ContentState: Codable, Hashable {
        var taskTitle: String
        var startedAt: Date
        var isTracking: Bool
    }
    // No static attributes needed for V1
}
```

**Lock Screen / Dynamic Island:**
- Task title (truncated)
- `Text(timerInterval: startedAt...Date.distantFuture, countsDown: false)` — ticks locally
- `.monospacedDigit()` to prevent layout jitter

**Watch Smart Stack (`.supplementalActivityFamilies([.small])`):**
- Task title (one line, truncated)
- Large `Text(timerInterval:)` elapsed timer
- No buttons (Watch `.small` surface is non-interactive)

**Interactive Stop Button:**
- `LiveActivityIntent` on the Live Activity
- Calls `POST /api/track` with `{taskId, action: "STOP"}` directly from Lock Screen
- No need to open the app

### Push Token Management

1. On app launch: observe `Activity<TimeTrackerAttributes>.pushToStartTokenUpdates`
2. Send push-to-start token to Go server via `POST /register`
3. When Live Activity starts (in-app or via push): observe `activity.pushTokenUpdates`
4. Send update token to Go server via `POST /register`
5. On token change: server invalidates old token, stores new one

### App Lifecycle

- **Foreground (`scenePhase == .active`):** Poll `GET /api/trackedItem` to reconcile state. If tracking started/stopped externally, update UI.
- **Background:** Live Activity + push handles everything. App does nothing.
- **Force-quit recovery:** Go server detects tracking is still active via polling, sends regular APNs alert: "Your timer is still running — tap to restore." User opens app, new Live Activity starts in-app.

### Offline Handling

- Persist current tracking state (task ID, title, start timestamp) to `@AppStorage`
- On next launch, restore UI from local state, then reconcile with API
- If network unavailable: show subtle inline banner, queue start/stop actions

### Error Handling

Three categories, handled inline (no modal alerts):

1. **Network unreachable:** Subtle banner, queue the action, retry on connectivity
2. **Invalid token:** Redirect to onboarding
3. **Rate limited:** Exponential backoff with "syncing..." indicator

### Secure Storage

- Marvin API token: iOS Keychain with `kSecAttrAccessibleAfterFirstUnlock`
- Go server URL: `@AppStorage` (not sensitive)
- Push tokens: managed by ActivityKit, sent to server over HTTPS

### Design Language

System utility aesthetic:
- Stock SwiftUI components
- System fonts (SF Pro)
- SF Symbols for icons
- Default material/translucency
- Automatic Light/Dark mode
- No custom colors, no brand identity
- Accessibility support comes free from system components

---

## Marvin API Usage

### Authentication

- Header: `X-API-Token` for all endpoints used in V1
- `X-Full-Access-Token` not needed (V1 doesn't use `/doc` endpoints)
- Token stored in iOS Keychain + Go server env var

### Endpoints Used (V1)

| Endpoint | Used By | Frequency |
|---|---|---|
| `GET /api/trackedItem` | Go server (polling) | ~300-700/day |
| `GET /api/todayItems` | iOS app (on "Start Tracking") | ~5-20/day |
| `POST /api/track` | iOS app (start/stop) | ~5-10/day |
| `GET /api/me` | iOS app (token validation) | ~1-2/day |
| **Total** | | **~310-730/day** (under 1,440 cap) |

### Rate Limits

| Limit | Value | Impact |
|---|---|---|
| Queries (burst) | 1 every 3 seconds | Server polling respects this |
| Queries (daily) | 1,440/day | Budget manager prevents exhaustion |
| Item creation | 1/second | Not applicable to V1 |

### Webhook Events Used

| Event | Webhook URL | Purpose |
|---|---|---|
| `startTracking` | `https://{server}/webhook/start` | Detect tracking started |
| `stopTracking` | `https://{server}/webhook/stop` | Detect tracking stopped |

Separate URLs per event (recommended) since payloads contain no event type field.

---

## Key Risks and Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| Webhook delivery failures (silent, client-side) | Medium | Adaptive fallback polling catches missed events within 30-60s |
| Live Activity 8-hour cap | Low | Pre-emptive renewal at 7h45m via push-to-start |
| Marvin API rate limit exhaustion | Low | Quota-aware budget manager, adaptive polling |
| Push-to-start token not available | Low | In-app start as primary path; regular APNs alert as fallback |
| Go server downtime | Low | Stateless restart, immediate state reconstruction via API poll |
| Duplicate webhook deliveries (~9s apart) | Low | Composite dedup key collapses duplicates |
| Force-quit kills token observation | Low | Server detects via polling, sends regular push to prompt re-open |

---

## What V1 Does NOT Include

- Task search or quick-add
- Category browsing or inbox triage
- Dedicated Apple Watch app or complication
- Time entry history or editing
- Settings screen (token re-entry via long-press "sign out")
- Home screen Widget (Live Activity covers this)
- Multiple user support
- Any task mutation beyond start/stop tracking

---

## Verified Facts

The following claims have been verified against official documentation (March 2026):

### Marvin API
- Base URL: `https://serv.amazingmarvin.com/api` (confirmed via OpenAPI spec)
- Auth headers: `X-API-Token` and `X-Full-Access-Token` (confirmed)
- Rate limits: 1/sec creation, 1/3s queries, 1,440/day (confirmed via YAML spec)
- All time tracking endpoints confirmed stable
- No "list all tasks" endpoint exists (confirmed)
- `/markDone` requires `ApiToken` only, not `FullAccessToken` (corrected from original findings)

### Webhooks
- Client-side AJAX requests (confirmed by wiki: "sent as cross-origin AJAX requests from the client")
- 23 event types including `startTracking`/`stopTracking` (confirmed)
- CORS: OPTIONS must return 200 (confirmed)
- No retry mechanism, no delivery logs (confirmed by omission)
- Issue #63: intermittent delivery gaps during specific daily hours (confirmed, open)
- Issue #53: HTTP endpoints may silently fail (confirmed, disputed by contributor)

### iOS / ActivityKit
- Live Activity 8-hour cap: confirmed, unchanged through iOS 18
- ActivityKit push can start (iOS 17.2+) and update Live Activities: confirmed
- Watch Smart Stack mirroring: confirmed (watchOS 11, any Series 6+ watch — NOT Series 9+ only)
- `Text(timerInterval:countsDown:)` counting up: confirmed
- `supplementalActivityFamilies([.small])` for custom Watch layout: confirmed
- `stale-date` in APNs payload: confirmed (JSON field in `aps` dictionary)
- `NSSupportsLiveActivitiesFrequentUpdates`: confirmed Info.plist key
- `activityStateUpdates` AsyncSequence: confirmed (`.active`, `.stale`, `.ended`, `.dismissed`)
- iOS 18 update frequency throttle (5-15s): confirmed, does NOT affect `Text(timerInterval:)`
- APNs `liveactivity` push type requires p8 certificates (p12 not supported): confirmed

### Go Server
- `sideshow/apns2`: 4.6K stars, maintained, supports `PushTypeLiveActivity`
- Go `net/http` HTTP/2: native, documented, guaranteed
- Cross-compilation: single command, fully static binary
- No external runtime dependencies

---

## References

- [MarvinAPI GitHub](https://github.com/amazingmarvin/MarvinAPI)
- [Marvin API Wiki](https://github.com/amazingmarvin/MarvinAPI/wiki/Marvin-API)
- [Marvin Webhooks Wiki](https://github.com/amazingmarvin/MarvinAPI/wiki/Webhooks)
- [ActivityKit — Apple Developer](https://developer.apple.com/documentation/activitykit)
- [Starting Live Activities with Push — Apple Developer](https://developer.apple.com/documentation/activitykit/starting-and-updating-live-activities-with-activitykit-push-notifications)
- [Bring Live Activities to Apple Watch — WWDC24](https://developer.apple.com/videos/play/wwdc2024/10068/)
- [sideshow/apns2 — GitHub](https://github.com/sideshow/apns2)
- [rs/cors — GitHub](https://github.com/rs/cors)
