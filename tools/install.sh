#!/bin/sh
set -eu

if [ "$(uname -s)" != "Darwin" ]; then
    echo "cloudcc: this installer supports macOS; on Windows use tools/install.ps1" >&2
    exit 1
fi
tools_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
launcher="$tools_dir/bin/cloudcc"
# Validate platform, architecture and binary before changing user files.
"$launcher" --version
case "${SHELL:-/bin/zsh}" in
    */zsh)
        profile_dir=${ZDOTDIR:-$HOME}
        profile="$profile_dir/.zprofile"
        rc="$profile_dir/.zshrc"
        ;;
    */bash)
        profile="$HOME/.bash_profile"
        if [ ! -f "$profile" ]; then
            if [ -f "$HOME/.bash_login" ]; then profile="$HOME/.bash_login"
            elif [ -f "$HOME/.profile" ]; then profile="$HOME/.profile"
            fi
        fi
        rc="$HOME/.bashrc"
        ;;
    *) echo "cloudcc: supported macOS shells are zsh and bash" >&2; exit 1 ;;
esac
bin_dir="$HOME/.local/bin"
destination="$bin_dir/cloudcc"
if [ -e "$destination" ] || [ -L "$destination" ]; then
    if [ ! -L "$destination" ]; then
        echo "cloudcc: refusing to overwrite existing file: $destination" >&2
        exit 1
    fi
    case "$(readlink "$destination")" in
        */tools/bin/cloudcc|*/tools/bin-darwin-amd64/cloudcc|*/tools/bin-darwin-arm64/cloudcc) ;;
        *) echo "cloudcc: refusing to replace an unrelated symlink: $destination" >&2; exit 1 ;;
    esac
fi
mkdir -p "$bin_dir" "$(dirname -- "$profile")"
ln -sfn "$launcher" "$destination"
for startup in "$profile" "$rc"; do
    if ! grep -Fq '# CloudCC CLI PATH' "$startup" 2>/dev/null; then
        cat >> "$startup" <<'PATH_BLOCK'

# CloudCC CLI PATH
case ":$PATH:" in
    *":$HOME/.local/bin:"*) ;;
    *) export PATH="$HOME/.local/bin:$PATH" ;;
esac
# End CloudCC CLI PATH
PATH_BLOCK
    fi
done
printf 'Installed cloudcc: %s
' "$destination"
printf 'PATH configured in %s and %s.
' "$profile" "$rc"
printf 'Open a new terminal, or run in the current terminal:
'
printf '  export PATH="$HOME/.local/bin:$PATH"
'
