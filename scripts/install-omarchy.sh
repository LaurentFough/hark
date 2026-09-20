#!/usr/bin/env bash
set -euo pipefail

# Installs this checkout as the Omarchy plugin without the generated
# konradk/hark-plugin distribution repository.
#
# The repository root is already a valid plugin folder (manifest.json,
# Overlay.qml, plugin/, quickshell/). The only thing missing is the runtime,
# which is not committed (bin/ is gitignored). This script builds it locally
# and links the checkout into the Omarchy plugins directory.
#
# Usage: scripts/install-omarchy.sh [--dry-run] [--skip-build] [--restart-shell]
#
#   --dry-run        print what would happen and change nothing
#   --skip-build     reuse an existing bin/harkd and bin/harkctl
#   --restart-shell  run omarchy-restart-shell afterwards (Quickshell caches
#                    plugin QML, so edited QML only loads after a restart)
#
# An existing plugin folder is moved to a backup directory outside the plugins
# directory, never deleted: a second manifest with the same id would collide.

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plugins_dir="${OMARCHY_PLUGINS_DIR:-${HOME}/.config/omarchy/plugins}"
backup_dir="${HARK_PLUGIN_BACKUP_DIR:-${HOME}/.config/omarchy/plugins.backup}"

dry_run=0
skip_build=0
restart_shell=0

usage() {
  sed -n '/^# Usage:/,/^$/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

for arg in "$@"; do
  case "${arg}" in
    --dry-run) dry_run=1 ;;
    --skip-build) skip_build=1 ;;
    --restart-shell) restart_shell=1 ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: ${arg}" >&2
      usage >&2
      exit 2
      ;;
  esac
done

run() {
  if ((dry_run)); then
    printf '+'
    printf ' %q' "$@"
    printf '\n'
  else
    "$@"
  fi
}

command -v jq >/dev/null 2>&1 || {
  echo "jq is required to read manifest.json." >&2
  exit 1
}

plugin_id="$(jq -r '.id // empty' "${repo_root}/manifest.json")"
if [[ ! "${plugin_id}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ || "${plugin_id}" == *..* ]]; then
  echo "manifest.json has no usable plugin id." >&2
  exit 1
fi
target="${plugins_dir}/${plugin_id}"

if command -v omarchy-plugin-validate >/dev/null 2>&1; then
  omarchy-plugin-validate "${repo_root}"
fi

if ((skip_build)); then
  echo "Skipping build; using the existing runtime in ${repo_root}/bin"
else
  run "${repo_root}/scripts/build-plugin-runtime.sh"
fi

if ! ((dry_run)); then
  for binary in harkd harkctl; do
    [[ -x "${repo_root}/bin/${binary}" ]] || {
      echo "Missing ${repo_root}/bin/${binary}; run without --skip-build." >&2
      exit 1
    }
  done
fi

run mkdir -p "${plugins_dir}"

if [[ -e "${target}" || -L "${target}" ]]; then
  if [[ "$(readlink -f -- "${target}")" == "$(readlink -f -- "${repo_root}")" ]]; then
    echo "${target} already resolves to this checkout."
  elif [[ -L "${target}" ]]; then
    echo "Replacing symlink ${target} -> $(readlink -- "${target}")"
    run rm -- "${target}"
  else
    backup="${backup_dir}/${plugin_id}.$(date +%Y%m%d%H%M%S)"
    echo "Moving existing ${target} to ${backup}"
    run mkdir -p "${backup_dir}"
    run mv -- "${target}" "${backup}"
  fi
fi

if [[ "$(readlink -f -- "${target}" 2>/dev/null || true)" != "$(readlink -f -- "${repo_root}")" ]]; then
  run ln -s -- "${repo_root}" "${target}"
  echo "Linked ${target} -> ${repo_root}"
fi

# The systemd unit runs ~/.local/bin/harkd. When that is a symlink into the
# plugin's bin/, restarting the unit is enough to pick up the new binary.
if systemctl --user cat harkd.service >/dev/null 2>&1; then
  run systemctl --user restart harkd.service
else
  echo "No harkd.service user unit found; the plugin will start its bundled daemon."
fi

if ((restart_shell)); then
  run omarchy-restart-shell
else
  echo "Run omarchy-restart-shell (or re-run with --restart-shell) so the shell loads the new QML."
fi

echo "Hark plugin ${plugin_id} installed from ${repo_root}"
