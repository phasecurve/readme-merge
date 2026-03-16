#!/bin/sh
set -eu

REPO="phasecurve/readme-merge"
BINARY="readme-merge"

main() {
    os="$(detect_os)"
    arch="$(detect_arch)"
    version="$(fetch_latest_version)"

    if [ -z "$version" ]; then
        fatal "could not determine latest version"
    fi

    install_dir="$(resolve_install_dir "$os")"
    asset="${BINARY}_${version#v}_${os}_${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${version}/${asset}"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    log "Installing ${BINARY} ${version} (${os}/${arch})"
    download "$url" "${tmpdir}/${asset}"
    tar -xzf "${tmpdir}/${asset}" -C "$tmpdir"

    if [ ! -f "${tmpdir}/${BINARY}" ]; then
        fatal "binary not found in archive"
    fi

    install_binary "${tmpdir}/${BINARY}" "$install_dir"
    verify_path "$install_dir"

    log "Installed ${BINARY} to ${install_dir}/${BINARY}"
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       fatal "unsupported OS: $(uname -s). See install.ps1 for Windows." ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             fatal "unsupported architecture: $(uname -m)" ;;
    esac
}

fetch_latest_version() {
    api_url="https://api.github.com/repos/${REPO}/releases/latest"
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        response="$(curl -fsSL -H "Authorization: token ${GITHUB_TOKEN}" "$api_url" 2>/dev/null)" || fatal "failed to fetch latest release (authenticated). Check your GITHUB_TOKEN."
    else
        response="$(curl -fsSL "$api_url" 2>/dev/null)" || fatal "failed to fetch latest release. If this is a private repo, set GITHUB_TOKEN."
    fi
    echo "$response" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//'
}

resolve_install_dir() {
    if [ -n "${INSTALL_DIR:-}" ]; then
        echo "$INSTALL_DIR"
        return
    fi
    case "$1" in
        linux)  echo "${HOME}/.local/bin" ;;
        darwin) echo "/usr/local/bin" ;;
    esac
}

download() {
    target_url="$1"
    dest="$2"
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        curl -fsSL -H "Authorization: token ${GITHUB_TOKEN}" -H "Accept: application/octet-stream" -o "$dest" "$target_url"
    else
        curl -fsSL -o "$dest" "$target_url"
    fi

    if [ ! -f "$dest" ] || [ ! -s "$dest" ]; then
        fatal "download failed: ${target_url}"
    fi
}

install_binary() {
    src="$1"
    dest_dir="$2"

    mkdir -p "$dest_dir" 2>/dev/null || true

    if cp "$src" "${dest_dir}/${BINARY}" 2>/dev/null; then
        chmod +x "${dest_dir}/${BINARY}"
        return
    fi

    log "Permission denied for ${dest_dir}, trying with sudo"
    sudo mkdir -p "$dest_dir"
    sudo cp "$src" "${dest_dir}/${BINARY}"
    sudo chmod +x "${dest_dir}/${BINARY}"
}

verify_path() {
    case ":${PATH}:" in
        *:"$1":*) ;;
        *)
            warn "${1} is not in your PATH"
            shell_name="$(basename "${SHELL:-/bin/sh}")"
            case "$shell_name" in
                zsh)  rc="${HOME}/.zshrc" ;;
                bash) rc="${HOME}/.bashrc" ;;
                *)    rc="your shell config" ;;
            esac
            warn "Add it with: echo 'export PATH=\"${1}:\$PATH\"' >> ${rc}"
            ;;
    esac
}

log()  { printf '%s\n' "$*" >&2; }
warn() { printf 'warning: %s\n' "$*" >&2; }
fatal() { printf 'error: %s\n' "$*" >&2; exit 1; }

main
