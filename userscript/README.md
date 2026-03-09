# Marvin Relay Tracker Userscript

A Tampermonkey/Greasemonkey userscript that overlays tracking controls on the [Amazing Marvin](https://app.amazingmarvin.com) web UI, synced with the relay server via Server-Sent Events (SSE).

## Why?

When tracking is stopped from the iOS app, the Marvin web UI still shows tracking as active due to localStorage desync. This userscript eliminates that problem by communicating directly with the relay server and receiving real-time state updates.

## Features

- **Real-time sync** — SSE connection to the relay server for instant state updates
- **Floating panel** — Shows current tracking status, task title, and elapsed time
- **Start buttons** — Injected into task elements for one-click tracking
- **Stop control** — Stop tracking from the overlay panel
- **Optimistic UI** — Immediate visual feedback, confirmed by server events
- **Fallback polling** — Polls `/status` every 5s when SSE is unavailable
- **Native button hiding** — Optionally hide Marvin's built-in tracking UI
- **Shadow DOM** — Fully isolated styles, no conflicts with Marvin's UI

## Installation

1. Install [Tampermonkey](https://www.tampermonkey.net/) (Chrome/Firefox/Safari/Edge)
2. Navigate to `http://<your-relay-server>/userscript/marvin-relay-tracker.user.js` — Tampermonkey intercepts `.user.js` URLs and offers to install
3. Click **Install**
4. Navigate to `https://app.amazingmarvin.com`
5. The "Relay Tracker" panel appears in the bottom-right corner

Alternatively, open the `marvin-relay-tracker.user.js` file directly and drag it into the Tampermonkey dashboard.

## Updating

### Auto-update (recommended)

When the relay server has `EXTERNAL_URL` configured, the served script includes `@updateURL` and `@downloadURL` metadata pointing back to the server. Tampermonkey periodically checks these URLs and prompts you when a new version is available.

To adjust the check interval: Tampermonkey Dashboard → Settings → Script Update → **Check Interval**.

### One-click update

Bookmark the install URL (`http://<your-relay-server>/userscript/marvin-relay-tracker.user.js`). When a new version is deployed, visit the bookmark — Tampermonkey detects the higher `@version` and offers to update.

### Manual update

Open Tampermonkey Dashboard → select "Marvin Relay Tracker" → paste the new script content → **Save**.

> **Note:** Tampermonkey only triggers an update when `@version` increases. Each userscript change requires a version bump to be picked up.

## Configuration

On first run, the settings panel opens automatically.

### Relay Server URL

Set this to your relay server address (e.g., `http://192.168.1.100:8080`).

Default: `http://localhost:8080`

### Hide Native Buttons

Toggle to hide Marvin's built-in time tracking buttons, reducing confusion when using the relay tracker.

### Security: `@connect`

The userscript uses `@connect *` to allow connections to any host, since the relay server address varies. To restrict this:

1. Edit the metadata block in the script
2. Replace `@connect *` with your specific server, e.g., `@connect 192.168.1.100`

## How It Works

```
Marvin Web UI
  ├── Userscript injects ▶ buttons into task elements
  ├── Floating panel shows tracking state
  └── EventSource connects to relay server /events endpoint
        ├── Receives: state, tracking_started, tracking_stopped events
        └── Fallback: polls GET /status every 5s if SSE disconnects
```

### SSE Events

| Event | Description |
|-------|-------------|
| `state` | Full state snapshot (sent on initial connection) |
| `tracking_started` | Task tracking began (includes taskId, taskTitle, startedAt) |
| `tracking_stopped` | Task tracking ended (includes taskId) |

### API Calls (via GM.xmlHttpRequest)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/start` | Start tracking a task |
| `POST` | `/stop` | Stop tracking |
| `GET` | `/status` | Get current status (fallback polling) |

## Server Requirements

The relay server must have the SSE endpoint enabled (added in the same release as this userscript). The `/events` endpoint:

- Streams Server-Sent Events
- Sends an initial `state` event with the current tracking state
- Broadcasts `tracking_started` and `tracking_stopped` events in real time
- Sends keepalive comments every 30 seconds
- CORS is already configured to allow `*` origins

### Testing the SSE endpoint

```bash
curl -N http://localhost:8080/events
# Should receive: event: state, then keepalive comments every 30s
```

## Releasing a New Version

### Homebrew

Create a new release tag (e.g., `v0.x.0`) and update the Homebrew formula with the new version and SHA. The binary rebuild automatically includes the latest userscript via `go:embed` — no extra steps needed for the userscript specifically.

### Tampermonkey auto-update

Two things must be true for Tampermonkey to pick up a new version:

1. **`@version` must be bumped** in `marvin-relay-tracker.user.js`. Tampermonkey compares versions numerically — if the version doesn't increase, the update is ignored.
2. **`EXTERNAL_URL` must be set** on the relay server so that `@updateURL` and `@downloadURL` resolve to real URLs (not the `__RELAY_URL__` placeholder).

Once a user upgrades the server (e.g., `brew upgrade marvin-relay`) and restarts it, Tampermonkey detects the higher `@version` on its next periodic check and prompts the user to update.

Users who installed the script *before* auto-update support (pre-0.3.0) won't have `@updateURL` in their installed copy. They need a one-time manual reinstall from the server URL to get auto-update metadata — after that, future updates are automatic.

## Troubleshooting

- **Panel shows "Disconnected"** — Check that the relay server is running and the URL is correct in settings
- **No ▶ buttons on tasks** — The script waits for Marvin's DOM to load; try refreshing. Buttons appear on `div[data-item-id][data-item-type="task"]` elements.
- **SSE not connecting** — Verify CORS is enabled on the server. Check browser console for errors.
- **Settings not persisting** — Ensure Tampermonkey has storage permissions for the script
