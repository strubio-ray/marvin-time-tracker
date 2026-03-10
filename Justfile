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

# Bump version, update changelog, tag, and push (use --dry-run to preview)
release *ARGS='--auto':
    cog bump {{ARGS}}
