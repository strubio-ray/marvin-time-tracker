# Apple Developer Setup

## Prerequisites

- Apple Developer account ($99/year)
- Xcode 16+

## 1. Create App ID

1. Go to [Certificates, Identifiers & Profiles](https://developer.apple.com/account/resources/identifiers/list)
2. Click **+** to register a new identifier
3. Select **App IDs** > **App**
4. Description: `Marvin Time Tracker`
5. Bundle ID (Explicit): `com.strubio.MarvinTimeTracker`
6. Enable capabilities:
   - **Push Notifications**
7. Click **Register**

## 2. Create Widget Extension App ID

1. Register another App ID:
   - Bundle ID: `com.strubio.MarvinTimeTracker.widgets`
   - No special capabilities needed

## 3. Create App Group

1. Go to **Identifiers** > **App Groups**
2. Click **+**
3. Description: `Marvin Time Tracker`
4. Identifier: `group.com.strubio.MarvinTimeTracker`
5. Assign this group to both App IDs above

## 4. Create APNs Key (p8)

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

## 5. Configure Environment

Copy `.env.example` (project root) to `.env` and fill in the values from the previous steps:

| Variable | Source |
|---|---|
| `APNS_KEY_ID` | Key ID from step 4 (10 characters) |
| `APNS_TEAM_ID` | Team ID from [Membership](https://developer.apple.com/account/#/membership) |
| `APNS_PRIVATE_KEY_PATH` | Path to the .p8 file downloaded in step 4 |
| `DEVELOPMENT_TEAM` | Same as `APNS_TEAM_ID` |

Copy the .p8 file to your server directory.

## 6. Build and Install iOS App

### First time: install Ruby dependencies

```bash
cd ios
bundle install
```

### Generate project and sync signing

```bash
bundle exec fastlane setup
```

This runs `xcodegen generate` and fetches match-managed development certificates
and provisioning profiles from the private certificates repo.

### Build, install, and launch on device

```bash
bundle exec fastlane deploy
```

This builds the app, installs it on the connected iPhone, and launches it.

On first install, trust the developer certificate on iPhone: Settings > General > VPN & Device Management.
