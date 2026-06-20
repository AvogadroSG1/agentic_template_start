#!/usr/bin/env bats

setup() {
  repo_root="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  script_path="$repo_root/.claude/hooks/secret-scan.sh"
}

run_scan_command_with_json() {
  local command_line="$1"
  run bash -lc "jq -nc --arg command \"$command_line\" '{tool_input:{command:\$command}}' | \"$script_path\" scan-command"
}

run_scan_command_with_flag() {
  local command_line="$1"
  run "$script_path" scan-command --command "$command_line"
}

@test "D9 blocks dotenv paths from guard JSON regardless of binary" {
  run_scan_command_with_json "cat .env"

  [ "$status" -eq 2 ]
  [[ "$output" == *"BLOCKED [D9]"* ]]
  [[ "$output" == *".env"* ]]
}

@test "D9 blocks secret path tokens across supported command forms" {
  local commands=(
    "awk '1' config/.env"
    "git show HEAD:.env"
    "cat app.pem"
    "python3 script.py .aws/credentials"
    "cat ~/.ssh/id_ed25519"
    "git status && cat .env"
  )

  for command_line in "${commands[@]}"; do
    run_scan_command_with_flag "$command_line"

    [ "$status" -eq 2 ]
    [[ "$output" == *"BLOCKED [D9]"* ]]
  done
}

@test "D10 blocks bare environment dump verbs" {
  local commands=(
    "env"
    "/usr/bin/env"
    "printenv"
    "/usr/bin/printenv"
    "set"
    "export -p"
    "declare -xp"
    "compgen -v"
  )

  for command_line in "${commands[@]}"; do
    run_scan_command_with_flag "$command_line"

    [ "$status" -eq 2 ]
    [[ "$output" == *"BLOCKED [D10]"* ]]
  done
}

@test "D10 allows filtered environment output" {
  run_scan_command_with_flag "env | grep -v SECRET"

  [ "$status" -eq 0 ]
}

@test "D10 blocks filtered env-dump variants outside the explicit carve-out" {
  local commands=(
    "/usr/bin/env | grep -v SECRET"
    "printenv | grep -v SECRET"
    "/usr/bin/printenv | grep -v SECRET"
  )

  for command_line in "${commands[@]}"; do
    run_scan_command_with_flag "$command_line"

    [ "$status" -eq 2 ]
    [[ "$output" == *"BLOCKED [D10]"* ]]
  done
}

@test "carve-outs and clean commands pass" {
  local commands=(
    "cat .env.example"
    "vim .env.template"
    "cat config.sample"
    "cat README.md"
  )

  for command_line in "${commands[@]}"; do
    run_scan_command_with_flag "$command_line"

    [ "$status" -eq 0 ]
  done
}

@test "empty stdin is clean" {
  run bash -lc "\"$script_path\" scan-command"

  [ "$status" -eq 0 ]
}

@test "unknown subcommand exits with usage" {
  run "$script_path" no-such-mode

  [ "$status" -eq 64 ]
}
