# Tailscale Funnel Setup

Tailscale Funnel exposes your Go relay server to the public internet over HTTPS, which is required for Marvin webhooks.

## Prerequisites

1. Tailscale installed on your Mac Studio (Standalone variant recommended)
2. MagicDNS enabled in the [admin console](https://login.tailscale.com/admin/dns)
3. HTTPS certificates enabled on the DNS page

## 1. Enable Funnel in ACL Policy

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

## 2. Start Funnel

```bash
tailscale funnel --bg 8080
```

This proxies your local port 8080 and exposes it at:

```
https://<machine-name>.<tailnet-name>.ts.net
```

The `8080` is the local target port. The public-facing side is always port 443 by default.

## 3. Verify

```bash
tailscale funnel status
curl https://<machine-name>.<tailnet-name>.ts.net/status
```

You should see `{"status":"ok","tracking":false,...}`.

## Persistence

The `--bg` flag makes Funnel persist across reboots. The configuration is stored by the Tailscale daemon, not in a separate process. No additional launchd entry is needed.

To stop Funnel:

```bash
tailscale funnel --bg off
```

## Notes

- Funnel can only listen on ports 443, 8443, and 10000 (public side)
- The local target port can be anything (e.g., 8080)
- Auto-provisioned Let's Encrypt certificate, no manual TLS setup needed
- Mac Studio must not sleep (System Settings > Energy > Prevent automatic sleeping)
- Your public URL format: `https://<machine-name>.<tailnet-name>.ts.net`
