#!/usr/bin/env bats

setup() {
  repo_root="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  script_path="$repo_root/.claude/hooks/secret-scan.sh"
  guard_delegate_path="$repo_root/test/guard-delegates-secret-scan.sh"
}

run_scan_command_with_json() {
  local command_line="$1"
  run bash -lc "jq -nc --arg command \"$command_line\" '{tool_input:{command:\$command}}' | \"$script_path\" scan-command"
}

run_scan_command_with_flag() {
  local command_line="$1"
  run "$script_path" scan-command --command "$command_line"
}

setup_git_fixture() {
  fixture_dir="$BATS_TEST_TMPDIR/repo-$BATS_TEST_NUMBER"
  mkdir -p "$fixture_dir"
  git init "$fixture_dir" >/dev/null
  git -C "$fixture_dir" config user.name "Bats Tester"
  git -C "$fixture_dir" config user.email "bats@example.com"
}

stage_fixture_file() {
  local relative_path="$1"
  mkdir -p "$fixture_dir/$(dirname "$relative_path")"
  printf 'fixture\n' >"$fixture_dir/$relative_path"
  git -C "$fixture_dir" add "$relative_path"
}

run_scan_staged() {
  run bash -lc "cd \"$fixture_dir\" && \"$script_path\" scan-staged"
}

@test "D9 blocks dotenv paths from guard JSON regardless of binary" {
  run_scan_command_with_json "cat .env"

  [ "$status" -eq 2 ]
  [[ "$output" == *"BLOCKED [D9]"* ]]
  [[ "$output" == *".env"* ]]
}

@test "D9 blocks sensitive path tokens across supported command forms" {
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

@test "scan-staged blocks staged dotenv files" {
  setup_git_fixture
  stage_fixture_file ".env"

  run_scan_staged

  [ "$status" -eq 2 ]
  [[ "$output" == *"BLOCKED [D9]"* ]]
  [[ "$output" == *".env"* ]]
}

@test "scan-staged blocks staged key files" {
  setup_git_fixture
  stage_fixture_file "config/deploy.key"

  run_scan_staged

  [ "$status" -eq 2 ]
  [[ "$output" == *"BLOCKED [D9]"* ]]
  [[ "$output" == *"config/deploy.key"* ]]
}

@test "scan-staged allows example and template carve-outs" {
  setup_git_fixture
  stage_fixture_file ".env.example"
  stage_fixture_file "config/api.key.template"

  run_scan_staged

  [ "$status" -eq 0 ]
}

@test "scan-staged allows safe staged files and empty index" {
  setup_git_fixture
  stage_fixture_file "README.md"

  run_scan_staged
  [ "$status" -eq 0 ]

  git -C "$fixture_dir" reset >/dev/null
  run_scan_staged
  [ "$status" -eq 0 ]
}

@test "scan-staged exits cleanly outside a git repo" {
  fixture_dir="$BATS_TEST_TMPDIR/not-a-repo"
  mkdir -p "$fixture_dir"

  run_scan_staged

  [ "$status" -eq 0 ]
}

@test "guard-style wrapper delegates D9 and D10 checks to the shared scanner" {
  run bash -lc "jq -nc '{tool_input:{command:\"git show HEAD:.env\"}}' | \"$guard_delegate_path\""

  [ "$status" -eq 2 ]
  [[ "$output" == *"BLOCKED [D9]"* ]]

  run bash -lc "jq -nc '{tool_input:{command:\"printenv\"}}' | \"$guard_delegate_path\""

  [ "$status" -eq 2 ]
  [[ "$output" == *"BLOCKED [D10]"* ]]
}
