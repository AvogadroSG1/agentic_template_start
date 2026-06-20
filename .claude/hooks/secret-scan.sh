#!/usr/bin/env bash
set -euo pipefail

SECRET_PATTERNS=(
  ".env"
  ".env."
  ".pem"
  ".key"
  "id_rsa"
  "id_ed25519"
  "credentials"
  "secret"
  ".tfstate"
  ".ssh/"
  ".aws/"
  ".gnupg/"
)
EXEMPT_PATTERNS=(
  "*.example"
  "*.sample"
  "*.template"
)
ENV_DUMP_VERBS=(
  "env"
  "printenv"
  "set"
  "export"
  "declare"
  "typeset"
  "compgen"
)

usage() {
  echo "usage: $0 scan-command [--command <line>]" >&2
}

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s\n' "$value"
}

normalize_token() {
  local token="$1"
  token="${token#\"}"
  token="${token#\'}"
  token="${token#\(}"
  token="${token#\[}"
  token="${token%\'}"
  token="${token%\"}"
  token="${token%\)}"
  token="${token%\]}"
  token="${token%,}"
  token="${token%;}"
  printf '%s\n' "$token"
}

is_exempt() {
  local token
  token="$(normalize_token "$1")"

  for pattern in "${EXEMPT_PATTERNS[@]}"; do
    case "$token" in
      $pattern) return 0 ;;
    esac
  done

  return 1
}

matches_secret() {
  local token
  token="$(normalize_token "$1")"

  if [ -z "$token" ] || is_exempt "$token"; then
    return 1
  fi

  for pattern in "${SECRET_PATTERNS[@]}"; do
    case "$token" in
      *"$pattern"*) return 0 ;;
    esac
  done

  return 1
}

is_filtered_env_dump() {
  case "$(trim "$1")" in
    "env | grep -v SECRET") return 0 ;;
  esac

  return 1
}

command_basename() {
  local command_line first_token
  command_line="$(trim "$1")"
  first_token="${command_line%%[[:space:]]*}"
  basename "$first_token"
}

is_env_dump() {
  local command_line first_token_name
  command_line="$(trim "$1")"

  if [ -z "$command_line" ] || is_filtered_env_dump "$command_line"; then
    return 1
  fi

  first_token_name="$(command_basename "$command_line")"
  case "$command_line" in
    env|printenv|set) return 0 ;;
    export\ -p|declare\ -p|declare\ -x|declare\ -xp|declare\ -px|typeset\ -p|typeset\ -x|compgen\ -v) return 0 ;;
  esac
  case "$first_token_name" in
    env|printenv) return 0 ;;
  esac

  return 1
}

read_command_from_stdin() {
  local stdin_payload=""

  if [ ! -t 0 ]; then
    stdin_payload="$(cat)"
  fi

  if [ -z "$stdin_payload" ]; then
    return 0
  fi

  jq -r '.tool_input.command // empty' <<<"$stdin_payload"
}

scan_command() {
  local command_line=""

  if [ "${1:-}" = "--command" ]; then
    if [ $# -lt 2 ]; then
      usage
      return 64
    fi
    command_line="$2"
  elif [ $# -gt 0 ]; then
    usage
    return 64
  else
    command_line="$(read_command_from_stdin)"
  fi

  command_line="$(trim "$command_line")"
  if [ -z "$command_line" ]; then
    return 0
  fi

  if is_env_dump "$command_line"; then
    echo "BLOCKED [D10]: bare environment dump command '$command_line'" >&2
    return 2
  fi

  # Irreducible gaps remain documented by design: obfuscated interpreter paths
  # and raw git object reads without path tokens are outside this string scan.
  read -r -a tokens <<<"$command_line"
  for token in "${tokens[@]}"; do
    if matches_secret "$token"; then
      echo "BLOCKED [D9]: command references secret path '$token'" >&2
      return 2
    fi
  done

  return 0
}

main() {
  local mode="${1:-}"
  case "$mode" in
    scan-command)
      shift
      scan_command "$@"
      ;;
    *)
      usage
      return 64
      ;;
  esac
}

main "$@"
