# Amazing Marvin Setup

## 1. Get API Token

1. Open Amazing Marvin (web or desktop)
2. Go to **Strategies** (left sidebar)
3. Search for **API** and enable the API strategy
4. Go to **Settings** > **API** (or find it in strategy settings)
5. Copy your **API Token**

## 2. Configure Webhooks

1. In Marvin, go to **Strategies** > search for **Webhooks**
2. Enable the Webhooks strategy
3. Add two webhooks:

### Start Tracking Webhook
- **Event:** `startTracking`
- **URL:** `https://<your-server>.ts.net/webhook/start`

### Stop Tracking Webhook
- **Event:** `stopTracking`
- **URL:** `https://<your-server>.ts.net/webhook/stop`

Replace `<your-server>.ts.net` with your Tailscale Funnel URL.

## Important Notes

- Webhooks are client-side AJAX requests from whichever Marvin client is active
- The server must be reachable via HTTPS (HTTP webhooks may silently fail)
- Webhooks have no retry mechanism; the Go server's polling acts as fallback
- Marvin may fire duplicate webhooks ~9 seconds apart; the server deduplicates these

## 3. Configure Server

Set the API token in your server's environment:

```bash
MARVIN_API_TOKEN=<your-api-token>
```

## 4. Configure iOS App

1. Launch the app on your iPhone
2. Enter your Go relay server URL (e.g., `https://mac-studio.tailnet-name.ts.net`)
3. Enter your Marvin API Token
4. The app validates the token by calling `GET /api/me`

## Rate Limits

| Limit | Value |
|---|---|
| Burst queries | 1 every 3 seconds |
| Daily queries | 1,440/day |
| Item creation | 1/second |

The server's quota counter tracks polling calls against the daily limit.
