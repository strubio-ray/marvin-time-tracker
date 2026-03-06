# Canonical copy lives in the homebrew-tap repo: strubio-ray/homebrew-tap
# This file is kept as a development reference.
class MarvinRelay < Formula
  desc "Go relay server bridging Amazing Marvin webhooks to Apple Live Activities"
  homepage "https://github.com/strubio-ray/marvin-time-tracker"
  version "0.1.0"
  url "https://github.com/strubio-ray/marvin-time-tracker/archive/refs/tags/v#{version}.tar.gz"
  sha256 "UPDATE_WITH_ACTUAL_SHA256" # TODO: update after first release
  license "MIT"

  livecheck do
    url :stable
    strategy :github_latest
  end

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"marvin-relay", "./server"
  end

  service do
    run [opt_bin/"marvin-relay"]
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
