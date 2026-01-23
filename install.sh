#!/bin/sh
set -e

# Blaxel MCP Server Installer
# Downloads and installs blaxel-mcp-server binary for your platform
# and provides configuration instructions for Cursor and Claude Code

OWNER="blaxel-ai"
REPO="blaxel-mcp-server"
BINARY="blaxel-mcp-server"
BINDIR="${BINDIR:-$HOME/.local/bin}"
PREFIX="$OWNER/$REPO"

usage() {
  cat <<EOF
$0: download and install blaxel-mcp-server

Usage: $0 [version]
  where [version] is a version number from
  https://github.com/${OWNER}/${REPO}/releases
  If absent, defaults to latest stable release.

Options:
  -h, --help    Show this help message
  --skip-config Skip the MCP configuration step

Environment variables:
  BINDIR        Installation directory (default: ~/.local/bin)
  VERSION       Version to install (default: latest)

Examples:
  # Install latest version
  curl -fsSL https://raw.githubusercontent.com/${OWNER}/${REPO}/main/install.sh | sh

  # Install specific version
  curl -fsSL https://raw.githubusercontent.com/${OWNER}/${REPO}/main/install.sh | sh -s -- v1.0.0

  # Install to custom directory
  curl -fsSL https://raw.githubusercontent.com/${OWNER}/${REPO}/main/install.sh | BINDIR=/usr/local/bin sh

EOF
}

# Portable shell functions
is_command() {
  command -v "$1" > /dev/null
}

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  echo "$os"
}

uname_arch() {
  arch=$(uname -m)
  case $arch in
    aarch64)    arch="arm64" ;;
    x86_64)     arch="amd64" ;;
    x86-64)     arch="amd64" ;;
    x64)        arch="amd64" ;;
    amd64)      arch="amd64" ;;
    arm64)      arch="arm64" ;;
    armv8*)     arch="arm64" ;;
  esac
  echo "${arch}"
}

uname_arch_check() {
  arch=$(uname_arch)
  case "$arch" in
    amd64)     return 0 ;;
    arm64)     return 0 ;;
  esac
  echo "$0: architecture '$arch' is not supported. Supported: amd64, arm64"
  return 1
}

uname_os_check() {
  os=$(uname_os)
  case "$os" in
    darwin)    return 0 ;;
    linux)     return 0 ;;
    windows)   return 0 ;;
    mingw*)    return 0 ;;
    msys*)     return 0 ;;
  esac
  echo "$0: OS '$os' is not supported. Supported: darwin, linux, windows"
  return 1
}

mktmpdir() {
  test -z "$TMPDIR" && TMPDIR="$(mktemp -d)"
  mkdir -p "${TMPDIR}"
  echo "${TMPDIR}"
}

http_download() {
  local_file=$1
  source_url=$2
  header=$3
  headerflag=''
  destflag=''
  if is_command curl; then
    cmd='curl --fail -sSL'
    destflag='-o'
    headerflag='-H'
  elif is_command wget; then
    cmd='wget -q'
    destflag='-O'
    headerflag='--header'
  else
    echo "http_download: unable to find wget or curl"
    return 1
  fi
  if [ -z "$header" ]; then
    $cmd $destflag "$local_file" "$source_url"
  else
    $cmd $headerflag "$header" $destflag "$local_file" "$source_url"
  fi
}

github_api() {
  local_file=$1
  source_url=$2
  header=""
  http_download "$local_file" "$source_url" "$header"
}

github_last_release() {
  owner_repo=$1
  giturl="https://api.github.com/repos/${owner_repo}/releases"
  html=$(github_api - "$giturl")
  version=$(echo "$html" | grep "\"tag_name\":" | cut -f4 -d'"' | grep -v -E "(preview|alpha|beta|rc|dev|pre|snapshot|nightly|canary|experimental|unstable)" | head -n 1)
  test -z "$version" && return 1
  echo "$version"
}

hash_sha256() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha256sum; then
    hash=$(gsha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha256sum; then
    hash=$(sha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 256 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl dgst -sha256 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 2
  else
    echo "hash_sha256: unable to find command to compute sha-256 hash"
    return 1
  fi
}

hash_sha256_verify() {
  TARGET=$1
  checksums=$2
  if [ -z "$checksums" ]; then
    echo "hash_sha256_verify: checksum file not specified"
    return 1
  fi
  BASENAME=${TARGET##*/}
  want=$(grep "${BASENAME}" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)
  if [ -z "$want" ]; then
    echo "hash_sha256_verify: unable to find checksum for '${TARGET}' in '${checksums}'"
    return 1
  fi
  got=$(hash_sha256 "$TARGET")
  if [ "$want" != "$got" ]; then
    echo "hash_sha256_verify: checksum mismatch for '$TARGET'"
    echo "  expected: $want"
    echo "  got:      $got"
    return 1
  fi
}

# Detect shell and provide PATH instructions
setup_path_interactive() {
  local bin_path="$1"
  local shell_name=""
  local rc_file=""
  local rc_file_path=""

  # Skip PATH setup for system directories that are already in PATH
  case "$bin_path" in
    /usr/local/bin|/usr/bin|/bin)
      echo ""
      echo "${BINARY} was installed successfully to $bin_path"
      echo ""
      echo "Since $bin_path is already in your system PATH, no additional configuration is needed."
      return
      ;;
  esac

  # Detect shell
  if [ -n "$SHELL" ]; then
    shell_name=$(basename "$SHELL")
  fi

  # Determine the appropriate RC file based on shell
  case "$shell_name" in
    zsh)
      rc_file="~/.zshrc"
      rc_file_path="$HOME/.zshrc"
      ;;
    bash)
      if [ -f "$HOME/.bash_profile" ]; then
        rc_file="~/.bash_profile"
        rc_file_path="$HOME/.bash_profile"
      else
        rc_file="~/.bashrc"
        rc_file_path="$HOME/.bashrc"
      fi
      ;;
    fish)
      rc_file="~/.config/fish/config.fish"
      rc_file_path="$HOME/.config/fish/config.fish"
      ;;
    *)
      rc_file="~/.profile"
      rc_file_path="$HOME/.profile"
      shell_name="shell"
      ;;
  esac

  echo ""
  echo "${BINARY} was installed successfully to $bin_path"
  echo ""

  # Check if PATH is already configured
  if [ -f "$rc_file_path" ] && grep -q "${bin_path}" "$rc_file_path"; then
    echo "${bin_path} is already in your PATH via $rc_file"
    return
  fi

  # Check if already in PATH
  case ":$PATH:" in
    *":${bin_path}:"*)
      echo "${bin_path} is already in your PATH"
      return
      ;;
  esac

  # Check if running in CI environment
  if [ -n "$CI" ] || [ -n "$GITHUB_ACTIONS" ] || [ -n "$GITLAB_CI" ] || [ -n "$CIRCLECI" ]; then
    echo "Detected CI environment - skipping PATH modification."
    return
  fi

  echo "To use ${BINARY}, you need to add it to your PATH."
  echo ""

  local response=""
  if [ -t 0 ]; then
    printf "Do you want to automatically add ${BINARY} to your PATH by modifying $rc_file? [y/N] "
    read -r response
  elif [ -e /dev/tty ]; then
    printf "Do you want to automatically add ${BINARY} to your PATH by modifying $rc_file? [y/N] " > /dev/tty
    read -r response < /dev/tty
  else
    response="n"
  fi

  case "$response" in
    [yY]|[yY][eE][sS])
      if [ "$shell_name" = "fish" ]; then
        mkdir -p "$(dirname "$rc_file_path")"
        printf "\n# Added by %s installer\nset -gx PATH %s \$PATH\n" "$BINARY" "$bin_path" >> "$rc_file_path"
      else
        printf "\n# Added by %s installer\nexport PATH=\"%s:\$PATH\"\n" "$BINARY" "$bin_path" >> "$rc_file_path"
      fi
      echo "Added ${BINARY} to PATH in $rc_file"
      echo ""
      echo "To use ${BINARY} in your current shell, run:"
      echo "  source $rc_file"
      ;;
    *)
      echo "To add ${BINARY} to your PATH manually, run:"
      echo ""
      if [ "$shell_name" = "fish" ]; then
        echo "  echo 'set -gx PATH $bin_path \$PATH' >> $rc_file"
      else
        echo "  echo 'export PATH=\"$bin_path:\$PATH\"' >> $rc_file"
      fi
      echo ""
      echo "Then restart your shell or run: source $rc_file"
      ;;
  esac
}

# Configure MCP for agentic applications
setup_mcp_config() {
  local binary_path="$1"

  echo ""
  echo "=========================================="
  echo "MCP Configuration for Agentic Applications"
  echo "=========================================="
  echo ""

  # Check if running in CI
  if [ -n "$CI" ] || [ -n "$GITHUB_ACTIONS" ] || [ -n "$GITLAB_CI" ] || [ -n "$CIRCLECI" ]; then
    print_mcp_instructions "$binary_path"
    return
  fi

  echo "Select the application to configure:"
  echo "  1) Cursor"
  echo "  2) Claude Code"
  echo "  3) Show manual instructions"
  echo "  0) Skip"
  echo ""

  local choice=""
  if [ -t 0 ]; then
    printf "Enter your choice [0-3]: "
    read -r choice
  elif [ -e /dev/tty ]; then
    printf "Enter your choice [0-3]: " > /dev/tty
    read -r choice < /dev/tty
  else
    choice="0"
  fi

  case "$choice" in
    1)
      configure_cursor "$binary_path"
      ;;
    2)
      configure_claude_code "$binary_path"
      ;;
    3)
      print_mcp_instructions "$binary_path"
      ;;
    *)
      echo ""
      echo "Skipping MCP configuration."
      echo "You can configure it later by following the instructions in the README."
      ;;
  esac
}

generate_cursor_deeplink() {
  local binary_path="$1"
  local config_json="{\"command\":\"${binary_path}\",\"env\":{\"BL_API_KEY\":\"YOUR_API_KEY\",\"BL_WORKSPACE\":\"YOUR_WORKSPACE\"}}"

  # Base64 encode the config (use printf for portability - echo -n doesn't work in all shells)
  local encoded_config=""
  if is_command base64; then
    # Check if base64 supports -w flag (Linux) or not (macOS)
    if base64 --help 2>&1 | grep -q "\-w"; then
      encoded_config=$(printf '%s' "$config_json" | base64 -w 0)
    else
      encoded_config=$(printf '%s' "$config_json" | base64 | tr -d '\n')
    fi
  else
    echo "Warning: base64 command not found, cannot generate deeplink"
    return 1
  fi

  echo "cursor://anysphere.cursor-deeplink/mcp/install?name=blaxel&config=${encoded_config}"
}

configure_cursor() {
  local binary_path="$1"
  local os=$(uname_os)

  echo ""
  echo "Configuring Cursor..."

  # Generate the deeplink
  local deeplink=$(generate_cursor_deeplink "$binary_path")
  if [ -z "$deeplink" ]; then
    echo "Error: Could not generate Cursor deeplink"
    return 1
  fi

  # Open the deeplink based on OS
  case "$os" in
    darwin)
      open "$deeplink" 2>/dev/null
      ;;
    linux)
      xdg-open "$deeplink" 2>/dev/null || sensible-browser "$deeplink" 2>/dev/null
      ;;
    windows|mingw*|msys*)
      start "$deeplink" 2>/dev/null || cmd /c start "$deeplink" 2>/dev/null
      ;;
  esac

  echo "deeplink: $deeplink"
  echo ""
  echo "Cursor should now prompt you to install the Blaxel MCP server."
  echo ""
  echo "Don't forget to replace YOUR_API_KEY and YOUR_WORKSPACE with your actual values."
}

configure_claude_code() {
  local binary_path="$1"
  local config_file=""

  os=$(uname_os)
  case "$os" in
    darwin)
      config_file="$HOME/Library/Application Support/Claude/claude_desktop_config.json"
      ;;
    linux)
      config_file="$HOME/.config/Claude/claude_desktop_config.json"
      ;;
    windows|mingw*|msys*)
      config_file="$APPDATA/Claude/claude_desktop_config.json"
      ;;
  esac

  echo ""
  echo "Configuring Claude Code MCP settings..."

  config_dir=$(dirname "$config_file")
  mkdir -p "$config_dir"

  if [ -f "$config_file" ]; then
    echo "Found existing config at $config_file"
    echo "Please add the following to your mcpServers section:"
    echo ""
    print_claude_config "$binary_path"
  else
    cat > "$config_file" << EOFCONFIG
{
  "mcpServers": {
    "blaxel": {
      "command": "${binary_path}",
      "env": {
        "BL_API_KEY": "YOUR_API_KEY",
        "BL_WORKSPACE": "YOUR_WORKSPACE"
      }
    }
  }
}
EOFCONFIG
    echo "Created Claude Code MCP config at $config_file"
    echo ""
    echo "Don't forget to replace YOUR_API_KEY and YOUR_WORKSPACE with your actual values in $config_file"
  fi
}

print_cursor_config() {
  local binary_path="$1"
  cat << EOF
{
  "blaxel": {
    "command": "${binary_path}",
    "env": {
      "BL_API_KEY": "YOUR_API_KEY",
      "BL_WORKSPACE": "YOUR_WORKSPACE"
    }
  }
}
EOF
}

print_claude_config() {
  local binary_path="$1"
  cat << EOF
{
  "blaxel": {
    "command": "${binary_path}",
    "env": {
      "BL_API_KEY": "YOUR_API_KEY",
      "BL_WORKSPACE": "YOUR_WORKSPACE"
    }
  }
}
EOF
}

print_mcp_instructions() {
  local binary_path="$1"

  echo ""
  echo "=== CURSOR ==="
  echo ""

  # Generate and display the deeplink
  local deeplink=$(generate_cursor_deeplink "$binary_path")
  if [ -n "$deeplink" ]; then
    echo "One-click install (recommended):"
    echo "  $deeplink"
    echo ""
    echo "Or add manually to ~/.cursor/mcp.json:"
  else
    echo "Add to ~/.cursor/mcp.json:"
  fi
  echo ""
  cat << EOF
{
  "mcpServers": {
    "blaxel": {
      "command": "${binary_path}",
      "env": {
        "BL_API_KEY": "YOUR_API_KEY",
        "BL_WORKSPACE": "YOUR_WORKSPACE"
      }
    }
  }
}
EOF

  echo ""
  echo "=== CLAUDE CODE ==="
  echo ""
  echo "Add to claude_desktop_config.json:"
  echo "  - macOS: ~/Library/Application Support/Claude/claude_desktop_config.json"
  echo "  - Linux: ~/.config/Claude/claude_desktop_config.json"
  echo "  - Windows: %APPDATA%/Claude/claude_desktop_config.json"
  echo ""
  cat << EOF
{
  "mcpServers": {
    "blaxel": {
      "command": "${binary_path}",
      "env": {
        "BL_API_KEY": "YOUR_API_KEY",
        "BL_WORKSPACE": "YOUR_WORKSPACE"
      }
    }
  }
}
EOF

  echo ""
  echo "Replace YOUR_API_KEY and YOUR_WORKSPACE with your actual Blaxel credentials."
}

# Main installation function
execute() {
  SCRIPT_TMPDIR=$(mktmpdir)

  OS=$(uname_os)
  ARCH=$(uname_arch)

  # Handle windows variants
  case "$OS" in
    mingw*|msys*)
      OS="windows"
      ;;
  esac

  BINARY_NAME="${BINARY}-${OS}-${ARCH}"

  if [ "$OS" = "windows" ]; then
    ARCHIVE_NAME="${BINARY_NAME}.zip"
    BINARY_NAME="${BINARY_NAME}.exe"
  else
    ARCHIVE_NAME="${BINARY_NAME}.tar.gz"
  fi

  TARBALL_URL="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
  CHECKSUMS_URL="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/checksums.txt"

  echo "${PREFIX}: downloading ${ARCHIVE_NAME} from ${VERSION}"
  http_download "${SCRIPT_TMPDIR}/${ARCHIVE_NAME}" "$TARBALL_URL"

  # Download and verify checksums
  echo "${PREFIX}: verifying checksum"
  http_download "${SCRIPT_TMPDIR}/checksums.txt" "$CHECKSUMS_URL"
  hash_sha256_verify "${SCRIPT_TMPDIR}/${ARCHIVE_NAME}" "${SCRIPT_TMPDIR}/checksums.txt"

  # Extract
  echo "${PREFIX}: extracting archive"
  cd "${SCRIPT_TMPDIR}"
  if [ "$OS" = "windows" ]; then
    unzip -q "${ARCHIVE_NAME}"
  else
    tar -xzf "${ARCHIVE_NAME}"
  fi

  # Install
  echo "${PREFIX}: installing to ${BINDIR}"
  install -d "${BINDIR}"
  install "${BINARY_NAME}" "${BINDIR}/${BINARY}"

  # Convert to absolute path for config
  if [ "${BINDIR#/}" = "${BINDIR}" ]; then
    ABSOLUTE_BINDIR="$(cd "${BINDIR}" && pwd)"
  else
    ABSOLUTE_BINDIR="${BINDIR}"
  fi

  INSTALLED_BINARY="${ABSOLUTE_BINDIR}/${BINARY}"

  setup_path_interactive "$ABSOLUTE_BINDIR"

  # Skip config if requested
  if [ "$SKIP_CONFIG" != "1" ]; then
    setup_mcp_config "$INSTALLED_BINARY"
  fi

  echo ""
  echo "Installation complete!"
  echo ""
  echo "To verify installation, run:"
  echo "  ${BINARY} --version"
}

# Parse arguments
SKIP_CONFIG=0
VERSION=""

while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --skip-config)
      SKIP_CONFIG=1
      shift
      ;;
    v*)
      VERSION="$1"
      shift
      ;;
    *)
      VERSION="$1"
      shift
      ;;
  esac
done

# Get latest version if not specified
if [ -z "${VERSION}" ]; then
  echo "${PREFIX}: checking GitHub for latest version"
  VERSION=$(github_last_release "$OWNER/$REPO")
  if [ -z "$VERSION" ]; then
    echo "${PREFIX}: unable to determine latest version"
    echo "Please specify a version: $0 v1.0.0"
    exit 1
  fi
  echo "${PREFIX}: found latest version ${VERSION}"
fi

# Validate environment
uname_os_check
uname_arch_check

# Run installation
execute
