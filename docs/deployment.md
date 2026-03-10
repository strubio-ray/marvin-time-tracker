# Deployment Guide

Complete walkthrough for deploying the Marvin Time Tracker on a Mac Studio.

## Overview

```
Mac Studio
├── marvin-relay (Go binary via brew services)
├── Tailscale Funnel (HTTPS proxy to port 8080)
└── .env (API tokens and APNs keys)

iPhone
└── MarvinTimeTracker.app (sideloaded via Xcode or TestFlight)
```

## Prerequisites

- Apple Developer account ($99/year)
- Xcode 16+
- Tailscale installed on your Mac Studio (Standalone variant recommended)
- Amazing Marvin account (web or desktop)

## Step 1: Create App IDs

1. Go to [Certificates, Identifiers & Profiles](https://developer.apple.com/account/resources/identifiers/list)
2. Click **+** to register a new identifier
3. Select **App IDs** > **App**
4. Description: `Marvin Time Tracker`
5. Bundle ID (Explicit): `com.strubio.MarvinTimeTracker`
6. Enable capabilities:
   - **Push Notifications**
7. Click **Register**

Register a second App ID for the widget extension:
- Bundle ID: `com.strubio.MarvinTimeTracker.widgets`
- No special capabilities needed

## Step 2: Create App Group

1. Go to **Identifiers** > **App Groups**
2. Click **+**
3. Description: `Marvin Time Tracker`
4. Identifier: `group.com.strubio.MarvinTimeTracker`
5. Assign this group to both App IDs above

## Step 3: Create APNs Key (p8)

1. Go to [Keys](https://developer.apple.com/account/resources/authkeys/list)
2. Click **+** to register a new key
3. Key Name: `Marvin APNs`
4. Enable **Apple Push Notifications service (APNs)**
5. Click **Continue**
6. Configure the two dropdown settings:
   - **Environment:** `Sandbox & Production`
   - **Key Restriction:** `Team Scoped (All Topics)`
7. Click **Register**
8. **Download the .p8 file** (you can only download it once)
9. Note the **Key ID** (10 characters, shown on the key details page)
10. Note your **Team ID** (shown in [Membership](https://developer.apple.com/account/#/membership))

## Step 4: Get Marvin API Token

1. Open Amazing Marvin (web or desktop)
2. Go to **Strategies** (left sidebar)
3. Search for **API** and enable the API strategy
4. Go to **Settings** > **API** (or find it in strategy settings)
5. Copy your **API Token**

## Step 5: Tailscale Funnel

Tailscale Funnel exposes the Go relay server to the public internet over HTTPS, which is required for Marvin webhooks.

### Enable Funnel in ACL Policy

In the [Tailscale admin console](https://login.tailscale.com/admin/acls), add or verify the `nodeAttrs` section includes Funnel:

```json
{
  "nodeAttrs": [
    {
      "target": ["autogroup:member"],
      "attr": ["funnel"]
    }
  ]
}
```

Ensure [MagicDNS](https://login.tailscale.com/admin/dns) and HTTPS certificates are enabled.

### Start Funnel

```bash
tailscale funnel --bg 8080
```

This proxies local port 8080 and exposes it at `https://<machine-name>.<tailnet-name>.ts.net`.

The `--bg` flag makes Funnel persist across reboots. To stop: `tailscale funnel --bg off`.

### Verify Funnel

```bash
tailscale funnel status
```

Notes:
- Funnel listens on port 443 (public side); the local target can be anything (e.g., 8080)
- Auto-provisioned Let's Encrypt certificate, no manual TLS setup needed
- Mac Studio must not sleep (System Settings > Energy > Prevent automatic sleeping)

## Step 6: Install and Configure Server

### Option A: Homebrew (recommended)

```bash
brew tap strubio-ray/tap
brew install marvin-relay
```

A default config file is installed to `/opt/homebrew/etc/marvin-relay/config`.

### Option B: Build from source

```bash
git clone https://github.com/strubio-ray/marvin-time-tracker.git
cd marvin-time-tracker
just build
cp server/config.example server/config
```

### Configure

Edit the config file with values from previous steps:

| Variable | Source |
|---|---|
| `MARVIN_API_TOKEN` | API token from Step 4 |
| `MARVIN_FULL_ACCESS_TOKEN` | Full Access Token from Step 4 |
| `APNS_KEY_ID` | Key ID from Step 3 (10 characters) |
| `APNS_TEAM_ID` | Team ID from [Membership](https://developer.apple.com/account/#/membership) |
| `APNS_KEY_P8_PATH` | Path to the .p8 file downloaded in Step 3 |
| `API_KEY` | Generate with `openssl rand -hex 32` — used by the iOS app to authenticate with the server |

Copy the .p8 file to the appropriate directory:
- **Homebrew**: `/opt/homebrew/etc/marvin-relay/`
- **From source**: project root (or wherever `APNS_KEY_P8_PATH` points)

### Start the server

```bash
# Homebrew
brew services start marvin-relay

# From source
just run
```

## Step 7: Verify Server

```bash
curl https://<your-machine>.ts.net/status
# Should return: {"status":"ok","tracking":false,...}
```

## Step 8: Configure Marvin Webhooks

Now that the server is running and reachable, configure Marvin to send webhooks:

1. In Marvin, go to **Strategies** > search for **Webhooks**
2. Enable the Webhooks strategy
3. Add two webhooks:

| Event | URL |
|---|---|
| `startTracking` | `https://<your-machine>.ts.net/webhook/start` |
| `stopTracking` | `https://<your-machine>.ts.net/webhook/stop` |

Replace `<your-machine>.ts.net` with your Tailscale Funnel URL.

Notes:
- Webhooks are client-side AJAX requests from whichever Marvin client is active
- The server must be reachable via HTTPS (HTTP webhooks may silently fail)
- Marvin may fire duplicate webhooks ~9 seconds apart; the server deduplicates these

## Step 9: Code Signing (Fastlane Match)

Certificates and provisioning profiles are managed by [Fastlane Match](https://docs.fastlane.tools/actions/match/) and stored in a private Git repo.

### Prerequisites

- Access to the certificates repo: `https://github.com/strubio-ray/fastlane-certificates`
  - Push access (SSH key or personal access token) for certificate creation
  - Read access is sufficient for `readonly` fetches
- Match manages profiles for both bundle IDs:
  - `com.strubio.MarvinTimeTracker`
  - `com.strubio.MarvinTimeTracker.widgets`

### App Store Connect API Key

Several Fastlane lanes require an App Store Connect API key. To set one up:

1. Go to [App Store Connect > Users and Access > Integrations > Team Keys](https://appstoreconnect.apple.com/access/integrations/api)
2. Click **+** to generate a new key
3. Name: `Fastlane` (or similar), Access: **Developer**
4. Download the `.p8` file and place it at `~/.private_keys/AuthKey.p8` (or any path)
5. Copy `ios/fastlane/.env.example` to `ios/fastlane/.env` and fill in:

| Variable | Source |
|---|---|
| `DEVELOPMENT_TEAM` | Same as `APNS_TEAM_ID` |
| `ASC_KEY_ID` | Key ID shown on the integrations page |
| `ASC_ISSUER_ID` | Issuer ID shown at the top of the integrations page |
| `ASC_KEY_P8_PATH` | Path to the downloaded .p8 file (default: `~/.private_keys/AuthKey.p8`) |

Fastlane auto-loads `ios/fastlane/.env`, so no manual sourcing is needed.

### Sync certificates

```bash
cd ios
bundle exec fastlane sync_certs
```

This fetches both development and App Store distribution certificates/profiles from the Match repo.

## Step 10: Install iOS App

### First time: install Ruby dependencies

```bash
cd ios
bundle install
```

### Build, install, and launch on device

```bash
just ios-deploy
```

This generates the Xcode project, syncs signing via match, builds, installs on the connected iPhone, and launches the app.

1. Enter server URL and API key in the app
2. Grant notification permissions

## Step 11: TestFlight Release (optional)

To distribute via TestFlight instead of sideloading:

```bash
just ios-testflight
```

This lane:
1. Queries TestFlight for the latest build number and auto-increments it
2. Syncs App Store distribution signing via Match
3. Builds a release IPA
4. Uploads to TestFlight
5. Tags the commit as `v{version}-{build_number}`

The app version is set in `ios/version.xcconfig` (`MARKETING_VERSION`). Only bump this when releasing a new user-facing version — build numbers are managed automatically.

To build without uploading (useful for verifying a release build compiles):

```bash
bundle exec fastlane build
```

## Step 12: Test End-to-End

1. Start tracking a task in Marvin (web or desktop)
2. Verify Live Activity appears on iPhone Lock Screen and Dynamic Island
3. Tap Stop on the Live Activity to stop tracking
4. Verify tracking stops in Marvin

## Troubleshooting

### Server not receiving webhooks
- Check Tailscale Funnel status: `tailscale funnel status`
- Verify HTTPS: `curl https://<your-machine>.ts.net/status`
- Check server logs: `cat /opt/homebrew/var/log/marvin-relay.log`

### Live Activity not appearing
- Ensure push-to-start token is registered: check `/status` endpoint for `hasPushToStartToken: true`
- Verify APNs configuration (Key ID, Team ID, p8 path)
- Check server logs for APNs errors

### Mac Studio goes to sleep
- System Settings > Energy > Prevent automatic sleeping when display is off

