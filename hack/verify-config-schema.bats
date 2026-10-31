#!/usr/bin/env bats
# Tests for hack/verify-config-schema.sh
#
# The script resolves its root from its own path and shells out to `go run`, so
# each test copies it into a temporary tree and puts a fake `go` on PATH that
# writes the files from generated/ in place of the real generator.

setup() {
  TEST_DIR="$(mktemp -d)"
  mkdir -p "${TEST_DIR}/hack" "${TEST_DIR}/chart" "${TEST_DIR}/generated/chart" "${TEST_DIR}/bin"
  cp "${BATS_TEST_DIRNAME}/verify-config-schema.sh" "${TEST_DIR}/hack/"

  cat >"${TEST_DIR}/bin/go" <<'EOF'
#!/usr/bin/env bash
cp -R generated/chart/. chart/
EOF
  chmod +x "${TEST_DIR}/bin/go"
  PATH="${TEST_DIR}/bin:${PATH}"

  echo "replicas: 1" >"${TEST_DIR}/chart/values.yaml"
  echo '{"type": "object"}' >"${TEST_DIR}/chart/values.schema.json"
  cp "${TEST_DIR}/chart/"* "${TEST_DIR}/generated/chart/"
}

teardown() {
  rm -rf "${TEST_DIR}"
}

@test "passes when the generator changes nothing" {
  run "${TEST_DIR}/hack/verify-config-schema.sh"

  [ "$status" -eq 0 ]
  [[ "$output" == *"config schema is up to date."* ]]
}

@test "passes on an uncommitted regen, since it compares before and after the generator" {
  git -C "${TEST_DIR}" init -q
  git -C "${TEST_DIR}" add chart
  git -C "${TEST_DIR}" -c user.name=test -c user.email=test@example.com commit -qm base
  echo "replicas: 2" >"${TEST_DIR}/chart/values.yaml"
  echo "replicas: 2" >"${TEST_DIR}/generated/chart/values.yaml"

  run "${TEST_DIR}/hack/verify-config-schema.sh"

  [ "$status" -eq 0 ]
}

@test "fails with the diff of only the file the generator changed" {
  echo '{"type": "object", "required": ["replicas"]}' >"${TEST_DIR}/generated/chart/values.schema.json"

  run "${TEST_DIR}/hack/verify-config-schema.sh"

  [ "$status" -eq 1 ]
  [[ "$output" == *'+{"type": "object", "required": ["replicas"]}'* ]]
  [[ "$output" == *"chart/values.schema.json (generated)"* ]]
  [[ "$output" != *"chart/values.yaml (generated)"* ]]
  [[ "$output" == *"Run 'just generate-config-schema'"* ]]
}

@test "fails when a hand edit to chart/values.yaml is reverted by the generator" {
  echo "replicas: 3" >"${TEST_DIR}/chart/values.yaml"

  run "${TEST_DIR}/hack/verify-config-schema.sh"

  [ "$status" -eq 1 ]
  [[ "$output" == *"-replicas: 3"* ]]
  [[ "$output" == *"+replicas: 1"* ]]
}

@test "works when called from outside its root" {
  cd /

  run "${TEST_DIR}/hack/verify-config-schema.sh"

  [ "$status" -eq 0 ]
}
