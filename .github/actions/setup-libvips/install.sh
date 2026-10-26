#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=/dev/null
source /etc/os-release
if [[ "$ID" != ubuntu || "$(dpkg --print-architecture)" != amd64 ]]; then
  echo "::error::libvips setup requires an Ubuntu amd64 runner."
  exit 1
fi

: "${VIPS_APT_CACHE:?}"
# RUNNER_TEMP can have private ancestors on hosted runners. APT only uses
# this separate /tmp tree; the Actions cache remains in RUNNER_TEMP.
work_dir=$(mktemp -d /tmp/armadacms-libvips.XXXXXX)
trap 'sudo rm -rf -- "$work_dir"' EXIT
archives="$work_dir/archives"
mkdir -p "$archives/partial" "$VIPS_APT_CACHE"
chmod 755 "$work_dir" "$archives"
# Stage complete packages only; APT manages its own locks and partial files.
find "$VIPS_APT_CACHE" -maxdepth 1 -type f -name '*.deb' -exec cp -t "$archives" -- {} +

apt_options=(
  -o "Dir::Etc::sourcelist=$work_dir/ubuntu.sources"
  -o "Dir::Etc::sourceparts=-"
  -o "Dir::State::lists=$work_dir/lists"
  -o "Dir::Cache::archives=$archives"
  -o APT::Keep-Downloaded-Packages=true
  -o Acquire::Retries=1
  -o Acquire::http::Timeout=15
  -o Acquire::https::Timeout=15
  -o APT::Update::Error-Mode=any
)

download_from() {
  local archive=$1 security=$2
  # Isolate sources and indexes from unrelated preinstalled runner repositories.
  mkdir -p "$work_dir/lists/partial"
  chmod 755 "$work_dir/lists"
  cat > "$work_dir/ubuntu.sources" <<EOF
Types: deb
URIs: $archive
Suites: $VERSION_CODENAME ${VERSION_CODENAME}-updates ${VERSION_CODENAME}-backports
Components: main restricted universe multiverse
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: $security
Suites: ${VERSION_CODENAME}-security
Components: main restricted universe multiverse
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg
EOF

  echo "Updating package indexes from $archive"
  if ! sudo timeout --signal=TERM --kill-after=10s 45s apt-get "${apt_options[@]}" update; then
    return 1
  fi
  echo "Downloading libvips packages from $archive (90 second limit)"
  # Only downloads are timed out; dpkg never runs inside a timeout.
  if ! sudo timeout --signal=TERM --kill-after=10s 90s apt-get "${apt_options[@]}" \
    install --download-only -y --no-install-recommends libvips-dev; then
    return 1
  fi
}

started=$SECONDS
if ! download_from http://azure.archive.ubuntu.com/ubuntu/ http://azure.archive.ubuntu.com/ubuntu/; then
  echo "::warning::Primary Ubuntu mirror failed or exceeded its time limit; trying the official archive."
  # Complete cached packages remain usable. APT refreshes indexes for the new URIs.
  if ! download_from https://archive.ubuntu.com/ubuntu/ https://security.ubuntu.com/ubuntu/; then
    echo "::error::Unable to download libvips packages from either Ubuntu mirror."
    exit 1
  fi
fi

# A cache hit still goes through APT dependency resolution and installation.
sudo apt-get "${apt_options[@]}" install --no-download -y --no-install-recommends libvips-dev
# Copy packages back as the runner user so actions/cache can save them.
find "$archives" -maxdepth 1 -type f -name '*.deb' -exec cp -t "$VIPS_APT_CACHE" -- {} +
echo "libvips $(pkg-config --modversion vips) installed in $((SECONDS - started)) seconds."
