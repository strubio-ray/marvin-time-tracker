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
5. Click **Continue** > **Register**
6. **Download the .p8 file** (you can only download it once)
7. Note the **Key ID** (10 characters, shown on the key details page)
8. Note your **Team ID** (shown in [Membership](https://developer.apple.com/account/#/membership))

## 5. Configure Server

Copy the p8 file to your server and set environment variables:

```bash
APNS_KEY_ID=<your key ID>
APNS_TEAM_ID=<your team ID>
APNS_PRIVATE_KEY_PATH=./AuthKey_<key-id>.p8
APNS_BUNDLE_ID=com.strubio.MarvinTimeTracker
```

## 6. Build and Install iOS App

```bash
cd ios
xcodegen generate
open MarvinTimeTracker.xcodeproj
```

1. Select your development team in Xcode signing settings
2. Connect your iPhone
3. Build and run (Cmd+R)
4. Trust the developer certificate on iPhone: Settings > General > VPN & Device Management
