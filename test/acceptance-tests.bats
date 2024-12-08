#!/usr/bin/env bats

# Environment Variables:
# Required:
# VAULT_PLUGIN_DIR  Location of the directory Vault should use to find all plugins.
#                   These plugins will need to be built for Linux.
#
# Optional:
# ===== Vault ===========================================
# VAULT_DOCKER_NAME  Name of the container to run Vault in
#                    Default: "vault-tests"
# VAULT_BIN          Location of the Vault binary. This is used to run queries against Vault.
#                    Default: "vault" (uses $PATH)
# VAULT_VERSION      Version of Vault to run in Docker
#                    Default: "latest"
# VAULT_TOKEN        The root token to use with Vault. These tests will not store the token
#                    in the local file system unlike the default behavior of a dev Vault server
#                    Default: "root-token"
# VAULT_PORT         The port number for Vault to run at. This is used to construct the
#                    $VAULT_ADDR environment variable. The IP address nor protocol are specified
#                    because it is assumed that http://127.0.0.1 will be used.
#                    Default: 8200
#
# ===== OpenLDAP ===========================================
# NXR_DOCKER_NAME    Name of the container to run Nexus Repository in
#                    Default: "nxr-tests"
# NXR_VERSION        Version of the Nexus Repository container to use
#                    Default: "latest"
# NXR_PORT           The port number for Nexus Repository to run at.
#                    Default: 8400
#
# ===== Docker ===========================================
# DOCKER_NETWORK  Name of the docker network to create
#                 Default: "vault-nexus-acceptance-tests-network"

# Required
vault_plugin_dir=${VAULT_PLUGIN_DIR:-"./dist/bin"}

# Optional
# docker_network=${DOCKER_NETWORK:-"vault-nexus-acceptance-tests-network"}

vault=${VAULT_BIN:-"vault"} # Uses $PATH

# vault_docker_name=${VAULT_DOCKER_NAME:-"vault-tests"}
# vault_version=${VAULT_VERSION:-"latest"}
vault_port=${VAULT_PORT:-"8200"}
vault_server_addr=${VAULT_SERVER_ADDR:-"127.0.0.1"}
export VAULT_ADDR="http://${vault_server_addr}:${vault_port}"
export VAULT_TOKEN=${VAULT_TOKEN:-"root-token"}

nxr_docker_name=${NXR_DOCKER_NAME:-"nxr-tests"}
# nxr_version=${NXR_VERSION:-"latest"}
nxr_server_addr=${NXR_SERVER_ADDR:-"127.0.0.1"}
nxr_port=${NXR_PORT:-"8400"}
nxr_admin_password="admin123"

##
log() {
  printf "# $(date) - %s\n" "${1}" >&3
}

if [ "${vault_plugin_dir}" == "" ]; then
  log "No plugin directory specified"
  exit 1
fi

##
wait_for_nxr(){
  log "[NXR] Waiting for Nexus Repository instance start..."
  until (curl -sfI -X GET "http://${nxr_server_addr}:${nxr_port}/service/rest/v1/status/writable" | grep -q 'HTTP/1.1 200 OK'); do
    printf "."
    sleep 2
  done

  log "[NXR] Verifying API status"
  if (curl -sfI -X GET --user "admin:${nxr_admin_password}" "http://${nxr_server_addr}:${nxr_port}/service/rest/v1/status/check"); then
    log "[NXR] Ready!"
  else
    log "[NXR] Could not verify that Nexus Repository API worked, please see the error above and check again!"
  fi
}

##
wait_for_vault(){
  log "[VAULT] Waiting for vault to become available..."
  until ( ${vault} status -address="${VAULT_ADDR}" ); do
    printf "."
    sleep 2
  done
  log "[VAULT] Ready!"
}

##
setup_file() {
  docker compose -f test/docker-compose.yml down
  docker compose -f test/docker-compose.yml up -d

  wait_for_vault
  wait_for_nxr

  # vault plugin register \
  #   -sha256="$(sha256sum ${VAULT_PLUGIN_DIR}/vault-plugin-secrets-nexus-repository | cut -d ' ' -f1)" \
  #   -command="vault-plugin-secrets-nexus-repository" \
  #   secret nexus

  vault secrets enable -path nexus vault-plugin-secrets-nexus-repository
}

##
teardown_file() {
  log "Tearing down containers..."
  docker compose -f test/docker-compose.yml down
  log "Teardown complete"
}

##
setup() {
  vault write nexus/config/admin username="admin" password="${nxr_admin_password}" url="http://${nxr_docker_name}:8081" # Nexus listen port is hardcoded to 8081
}

##
teardown() {
  # Remove any roles that were created so they don't bleed over to other tests
  output=$(vault list -format=json nexus/roles || true) # "or true" so it doesn't show an error if there are no roles

  roles=$(echo "${output}" | jq -r .[])
  for role in ${roles}; do
    vault delete "nexus/roles/${role}" > /dev/null
  done

  #
  vault delete nexus/config/admin
}

##
@test "Test role - Read/write/delete" {
  ttl=5
  max_ttl=10

  # Create role
  run vault write nexus/roles/test-role \
    nexus_roles="nx-anonymous,nx-admin" \
    ttl="${ttl}s" \
    max_ttl="${max_ttl}s" \
    user_id_template="{{ printf \"v-%s-%s\" (.DisplayName | truncate 64) (unix_time) | truncate 128 | lowercase }}" \
    user_email="test@email.org"

  [ ${status} -eq 0 ]

  # Read role and make sure it matches what we expect
  run vault read nexus/roles/test-role -format=json
  [ ${status} -eq 0 ]
  expected='{
    "max_ttl": 10,
    "name": "test-role",
    "nexus_roles": [
      "nx-anonymous",
      "nx-admin"
    ],
    "ttl": 5,
    "user_email": "test@email.org",
    "user_id_template": "{{ printf \"v-%s-%s\" (.DisplayName | truncate 64) (unix_time) | truncate 128 | lowercase }}"
  }'
  run jq --argjson a "${output}" --argjson b "${expected}" -n '$a.data == $b'
  [ ${status} -eq 0 ]
  [ "${output}" == "true" ]

  ## Delete the role and ensure that it and the creds endpoint isn't readable
  run vault delete nexus/roles/test-role
  [ ${status} -eq 0 ]

  run vault read nexus/roles/test-role
  [ ${status} -ne 0 ]

  run vault read nexus/creds/test-role
  [ ${status} -ne 0 ]
}

##
@test "Test role - List" {
  # Create a bunch of roles with different prefixes
  for id in $(seq -f "%02g" 0 10); do
    rolename="test-role-${id}"
    run vault write "nexus/roles/${rolename}" nexus_roles="nx-anonymous"
    [ ${status} -eq 0 ]

    rolename="role-test-${id}"
    run vault write "nexus/roles/${rolename}" nexus_roles="nx-anonymous"
    [ ${status} -eq 0 ]
  done

  # Test list
  run vault list -format=json nexus/roles
  [ ${status} -eq 0 ]

  expected='[
    "role-test-00",
    "role-test-01",
    "role-test-02",
    "role-test-03",
    "role-test-04",
    "role-test-05",
    "role-test-06",
    "role-test-07",
    "role-test-08",
    "role-test-09",
    "role-test-10",
    "test-role-00",
    "test-role-01",
    "test-role-02",
    "test-role-03",
    "test-role-04",
    "test-role-05",
    "test-role-06",
    "test-role-07",
    "test-role-08",
    "test-role-09",
    "test-role-10"
  ]'
  run jq --argjson a "${output}" --argjson b "${expected}" -n '$a == $b'
  [ ${status} -eq 0 ]
  [ "${output}" == "true" ]
}

##
@test "Test creds - Credential lifecycle without renewal" {
  ttl=5
  max_ttl=10

  # Create role
  run vault write nexus/roles/testrole \
    nexus_roles="nx-anonymous,nx-admin" \
    ttl="${ttl}s" \
    max_ttl="${max_ttl}s" \
    user_id_template="{{ printf \"v-%s-%s-%s\" (.RoleName) (.DisplayName | truncate 64) (unix_time) | truncate 128 | lowercase }}" \
    user_email="test@email.org"
  [ ${status} -eq 0 ]

  # Get credentials
  run vault read -format=json nexus/creds/testrole
  [ ${status} -eq 0 ]


  ## Assert all fields that should be there are there
  assertion=$(echo "${output}" | jq '.data | has("user_id")')
  [ "${assertion}" == "true" ]

  assertion=$(echo "${output}" | jq '.data | has("password")')
  [ "${assertion}" == "true" ]

  assertion=$(echo "${output}" | jq '.data | has("email_address")')
  [ "${assertion}" == "true" ]

  assertion=$(echo "${output}" | jq '.data | has("nexus_roles")')
  [ "${assertion}" == "true" ]

  ## Assert the fields are structured correctly
  user_id="$(echo "${output}" | jq -r '.data.user_id')"
  [[ "${user_id}" =~ ^v-testrole-token-[0-9]{10}$ ]]

  password="$(echo "${output}" | jq -r '.data.password')"
  [[ "${password}" =~ ^[a-zA-Z0-9]{64}$ ]]

  email="$(echo "${output}" | jq -r '.data.email_address')"
  [[ ${email} == "test@email.org" ]]

  nexus_role0="$(echo "${output}" | jq -r '.data.nexus_roles[0]')"
  [[ "${nexus_role0}" == "nx-anonymous" ]]
  nexus_role1="$(echo "${output}" | jq -r '.data.nexus_roles[1]')"
  [[ "${nexus_role1}" == "nx-admin" ]]

  ## Assert the credentials work in Nexus Repository
  run curl -sfI -X GET --user "${user_id}:${password}" "http://127.0.0.1:${nxr_port}/service/rest/v1/status/check"
  [ ${status} -eq 0 ]
  [[ "${output}" == *"200 OK"* ]]


  ## Assert the credentials no longer work after their TTL
  sleep $((ttl + 1))

  run curl -sfI -X GET --user "${user_id}:${password}" "http://127.0.0.1:${nxr_port}/service/rest/v1/status/check"
  [ ${status} -ne 0 ]
  [[ "${output}" == *"401 Unauthorized"* ]]
}

##
@test "Test creds - Credential lifecycle with renewal" {
  ttl=5
  max_ttl=10

  # Create role
  run vault write nexus/roles/testrole \
    nexus_roles="nx-anonymous,nx-admin" \
    ttl="${ttl}s" \
    max_ttl="${max_ttl}s" \
    user_id_template="{{ printf \"v-%s-%s-%s\" (.RoleName) (.DisplayName | truncate 64) (unix_time) | truncate 128 | lowercase }}" \
    user_email="test@email.org"
  [ ${status} -eq 0 ]

  # Get credentials
  run vault read -format=json nexus/creds/testrole
  [ ${status} -eq 0 ]

  #
  lease_id=$(echo "${output}" | jq -r .lease_id)

  ## Assert all fields that should be there are there
  assertion=$(echo "${output}" | jq '.data | has("user_id")')
  [ "${assertion}" == "true" ]

  assertion=$(echo "${output}" | jq '.data | has("password")')
  [ "${assertion}" == "true" ]

  assertion=$(echo "${output}" | jq '.data | has("email_address")')
  [ "${assertion}" == "true" ]

  assertion=$(echo "${output}" | jq '.data | has("nexus_roles")')
  [ "${assertion}" == "true" ]

  ## Assert the fields are structured correctly
  user_id="$(echo "${output}" | jq -r '.data.user_id')"
  [[ "${user_id}" =~ ^v-testrole-token-[0-9]{10}$ ]]

  password="$(echo "${output}" | jq -r '.data.password')"
  [[ "${password}" =~ ^[a-zA-Z0-9]{64}$ ]]

  email="$(echo "${output}" | jq -r '.data.email_address')"
  [[ ${email} == "test@email.org" ]]

  nexus_role0="$(echo "${output}" | jq -r '.data.nexus_roles[0]')"
  [[ "${nexus_role0}" == "nx-anonymous" ]]
  nexus_role1="$(echo "${output}" | jq -r '.data.nexus_roles[1]')"
  [[ "${nexus_role1}" == "nx-admin" ]]

  ## Assert the credentials work in Nexus Repository
  run curl -sfI -X GET --user "${user_id}:${password}" "http://127.0.0.1:${nxr_port}/service/rest/v1/status/check"
  [ ${status} -eq 0 ]
  [[ "${output}" == *"200 OK"* ]]


  ## Wait until the credentials have been around for a detectable amount of time
  wait_until_dur=$((ttl - 2))

  log "Waiting for a couple of seconds..."
  while true; do
    run vault write -format=json sys/leases/lookup lease_id="${lease_id}"
    [ ${status} -eq 0 ]
    _ttl=$(echo "${output}" | jq .data.ttl)
    if [ "${_ttl}" -le "${wait_until_dur}" ]; then
      break
    fi
    sleep 1
  done

  before_renewal=$(date +%s)

  log "Renewing lease..."
  run vault lease renew "${lease_id}"
  [ ${status} -eq 0 ]

  sleep_time=$(($(date +%s) - before_renewal + 1))

  ## Wait until after the original TTL but less than the new TTL
  log "Sleeping until after original TTL (${sleep_time}s)..."
  sleep $((sleep_time))

  run curl -sfI -X GET --user "${user_id}:${password}" "http://${nxr_server_addr}:${nxr_port}/service/rest/v1/status/check"
  [ ${status} -eq 0 ]
  [[ "${output}" == *"200 OK"* ]]

  log "Sleeping until lease expires"
  while true; do
    run vault write -format=json sys/leases/lookup lease_id="${lease_id}"
    if [ ${status} -ne 0 ]; then
      break
    fi
    sleep 1
  done

  run curl -sfI -X GET --user "${user_id}:${password}" "http://${nxr_server_addr}:${nxr_port}/service/rest/v1/status/check"
  [ ${status} -ne 0 ]
  [[ "${output}" == *"401 Unauthorized"* ]]
}

##
@test "Test config - Rotate admin password" {
  run vault write -f nexus/config/rotate
  [ ${status} -eq 0 ]

  run curl -sfI -X GET --user "admin:${nxr_admin_password}" "http://${nxr_server_addr}:${nxr_port}/service/rest/v1/status/check"
  [ ${status} -ne 0 ]
}
