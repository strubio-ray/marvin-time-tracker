# Deployment Guide

Complete walkthrough for deploying the Marvin Time Tracker on a Mac Studio.

## Overview

```
Mac Studio
├── marvin-relay (Go binary via brew services)
├── Tailscale Funnel (HTTPS proxy to port 8080)
└── .env (API tokens and APNs keys)

iPhone
└── MarvinTimeTracker.app (sideloaded via Xcode)
```

## Step 1: Apple Developer Setup

Follow [apple-developer-setup.md](./apple-developer-setup.md) to:
- Create App IDs and App Group
- Generate APNs p8 key
- Note your Key ID and Team ID

## Step 2: Marvin Setup

Follow [marvin-setup.md](./marvin-setup.md) to:
- Get your Marvin API token
- Configure webhooks (after Tailscale Funnel is running)

## Step 3: Tailscale Funnel

Follow [tailscale-funnel-setup.md](./tailscale-funnel-setup.md) to:
- Enable Funnel in your tailnet policy
- Expose port 8080 via HTTPS

## Step 4: Install Server

### Option A: Homebrew (recommended)

```bash
brew tap strubio/services
brew install marvin-relay
```

Create the environment file:

```bash
cp /usr/local/share/marvin-relay/.env.example ~/Library/Application\ Support/marvin-relay/.env
# Edit with your tokens and key paths
```

Start the service:

```bash
brew services start marvin-relay
```

### Option B: Build from source

```bash
git clone https://github.com/strubio/marvin-time-tracker.git
cd marvin-time-tracker
make build

# Copy .env.example and configure
cp server/.env.example .env
# Edit .env with your values

# Run
make run
```

## Step 5: Verify Server

```bash
curl https://<your-machine>.ts.net/status
# Should return: {"status":"ok","tracking":false,...}
```

## Step 6: Configure Marvin Webhooks

Now that the server is running and reachable, go back to Marvin and add the webhook URLs:

- `startTracking` -> `https://<your-machine>.ts.net/webhook/start`
- `stopTracking` -> `https://<your-machine>.ts.net/webhook/stop`

## Step 7: Install iOS App

```bash
cd ios
xcodegen generate
open MarvinTimeTracker.xcodeproj
```

1. Connect iPhone, select as run destination
2. Build and run
3. Enter server URL and Marvin API token in the app
4. Grant notification permissions

## Step 8: Test End-to-End

1. Start tracking a task in Marvin (web or desktop)
2. Verify Live Activity appears on iPhone Lock Screen and Dynamic Island
3. Tap Stop on the Live Activity to stop tracking
4. Verify tracking stops in Marvin

## Troubleshooting

### Server not receiving webhooks
- Check Tailscale Funnel status: `tailscale funnel status`
- Verify HTTPS: `curl https://<your-machine>.ts.net/status`
- Check server logs: `brew services log marvin-relay` or check log files

### Live Activity not appearing
- Ensure push-to-start token is registered: check `/status` endpoint for `hasPushToStartToken: true`
- Verify APNs configuration (Key ID, Team ID, p8 path)
- Check server logs for APNs errors

### Mac Studio goes to sleep
- System Settings > Energy > Prevent automatic sleeping when display is off

### Rate limit exceeded
- Check server logs for quota warnings
- Increase `POLL_INTERVAL_IDLE` to reduce call volume
