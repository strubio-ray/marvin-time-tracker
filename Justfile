# BEGIN agent-sandbox managed
mod? sandbox ".vm"
# END agent-sandbox managed

# Build automation for marvin-time-tracker

# List available recipes
default:
    @just --list

# Build server binary to server/marvin-relay
build:
    go build -o server/marvin-relay ./server

# Run all server tests
test:
    go test ./server/...

# Build and run server
run: build
    ./server/marvin-relay --config server/config

# Remove built binary
clean:
    rm -f server/marvin-relay

# Deploy with APNS_ENV=development
deploy-dev: (_deploy "development")

# Deploy with APNS_ENV=production
deploy-prod: (_deploy "production")

_deploy env: test build
    install -m 755 server/marvin-relay /opt/homebrew/opt/marvin-relay/bin/marvin-relay
    rm -f /opt/homebrew/var/log/marvin-relay.log
    brew services restart marvin-relay
    @echo "Deployed ({{env}}). Tailing logs (Ctrl-C to stop)..."
    @tail -f /opt/homebrew/var/log/marvin-relay.log

# Build, install, and launch on device via Fastlane
ios-deploy:
    cd ios && bundle exec fastlane deploy

# Build and upload to TestFlight
ios-testflight:
    cd ios && bundle exec fastlane testflight_release

# Bump userscript version (patch, minor, or major)
bump-userscript part='patch':
    #!/usr/bin/env bash
    set -euo pipefail
    file="userscript/marvin-relay-tracker.user.js"
    old=$(sed -n 's/.*@version[[:space:]]*\([0-9][0-9.]*\).*/\1/p' "$file")
    IFS='.' read -r major minor patch <<< "$old"
    case "{{part}}" in
        patch) patch=$((patch + 1)) ;;
        minor) minor=$((minor + 1)); patch=0 ;;
        major) major=$((major + 1)); minor=0; patch=0 ;;
        *) echo "Unknown part '{{part}}'. Use: patch, minor, or major"; exit 1 ;;
    esac
    new="${major}.${minor}.${patch}"
    sed -i'' -e "s/@version[[:space:]]*${old}/@version      ${new}/" "$file"
    echo "userscript: ${old} → ${new}"

# Bump version, update changelog, tag, and push (use --dry-run to preview)
release *ARGS='--auto':
    cog bump {{ARGS}}
