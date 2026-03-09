# Canonical copy lives in the homebrew-tap repo: strubio-ray/homebrew-tap
# This file is kept as a development reference.
class MarvinRelay < Formula
  desc "Go relay server bridging Amazing Marvin webhooks to Apple Live Activities"
  homepage "https://github.com/strubio-ray/marvin-time-tracker"
  version "0.1.0"
  url "https://github.com/strubio-ray/marvin-time-tracker/archive/refs/tags/v#{version}.tar.gz"
  sha256 "6094a89d0f0eb0b91ef05af4f8ef3b06707b4c0d54fc1732a538d51caf060012"
  license "MIT"

  livecheck do
    url :stable
    strategy :github_latest
  end

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"marvin-relay", "./server"
    (etc/"marvin-relay").install "server/config.example" => "config"
  end

  service do
    run [opt_bin/"marvin-relay", "--config", etc/"marvin-relay/config"]
    keep_alive true
    working_dir var/"marvin-relay"
    log_path var/"log/marvin-relay.log"
    error_log_path var/"log/marvin-relay.log"
    environment_variables PATH: std_service_path_env
  end

  def post_install
    (var/"marvin-relay").mkpath
  end

  test do
    assert_match "marvin-relay 0.1.0", shell_output("#{bin}/marvin-relay --version")
  end
end
