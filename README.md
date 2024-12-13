[![GitHub license](https://img.shields.io/github/license/manhtukhang/vault-plugin-secrets-nexus-repository.svg)](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/blob/main/LICENSE)
[![Release](https://img.shields.io/github/release/manhtukhang/vault-plugin-secrets-nexus-repository.svg)](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/releases/latest)
[![Lint](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/lint.yaml/badge.svg)](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/lint.yaml)
[![Unit Test](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/test.yaml/badge.svg)](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/test.yaml)
[![Acceptance Test](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/test-acceptance.yaml/badge.svg)](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/test-acceptance.yaml)
[![Security scanning](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/scan.yaml/badge.svg)](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/actions/workflows/scan.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/manhtukhang/vault-plugin-secrets-nexus-repository)](https://goreportcard.com/report/github.com/manhtukhang/vault-plugin-secrets-nexus-repository)
[![Maintainability](https://api.codeclimate.com/v1/badges/cc38296a9c2c3bb2c023/maintainability)](https://codeclimate.com/github/manhtukhang/vault-plugin-secrets-nexus-repository/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/cc38296a9c2c3bb2c023/test_coverage)](https://codeclimate.com/github/manhtukhang/vault-plugin-secrets-nexus-repository/test_coverage)

---
# Vault Secrets Plugin for Nexus Reporitory

This is a [(HashiCorp) Vault secrets plugin](https://developer.hashicorp.com/vault/docs/plugins) which talks to [(Sonatype) Nexus Repository](https://www.sonatype.com/products/sonatype-nexus-repository) server and will dynamically create/revoke local user with predefined role(s). This backend can be mounted multiple times to provide access to multiple Nexus Repository servers, and work with both Pro and OSS versions.

Using this plugin, you can limit the accidental exposure window of Nexus Repository user's credentials; useful for continuous integration servers.


---
## INSTALLATION

> [!IMPORTANT]
> This plugin is designed to be `run on Vault servers` and `could connect to Nexus Repository server(s) with high privileges`.
> Therefore, DO NOT use this (or any 3rd party plugins) on critical production environment without understanding about the security risks!
> Please read [SECURITY section](#security) carefully before any uses.


### Using pre-built releases

You can find pre-built releases of the plugin [here](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/releases) and download the latest binary file corresponding to your target OS.

### From Sources

If you prefer to build the plugin from sources, clone the GitHub repository locally and run the command `make build` from the root of the sources directory.

Upon successful compilation, the resulting `vault-plugin-secrets-nexus-repository` binary is stored in the `dist/bin` directory.

### Register plugin to Vault server

Copy the plugin binary into a location of your choice; this directory must be specified as the [`plugin_directory`](https://developer.hashicorp.com/vault/docs/configuration#plugin_directory) in the Vault configuration file:

```hcl
plugin_directory = "path/to/plugin/directory"
```

Register the plugin in the Vault server's [plugin catalog](https://developer.hashicorp.com/vault/docs/plugins/plugin-management):
```sh
$ vault plugin register \
  -sha256=$(sha256sum path/to/plugin/directory/vault-plugin-secrets-nexus-repository | cut -d " " -f 1) \
  -command=vault-plugin-secrets-nexus-repository \
  secret \
  nexus
```

> [!CAUTION]
> This inline checksum calculation above is provided for illustration purpose and does not validate your binary. It should **not** be used for production environment. Instead you should use the checksum provided in the release archive file as [part of the releases](https://github.com/manhtukhang/vault-plugin-secrets-nexus-repository/releases). See [Verify downloaded artifact from GitHub releases](#verify-downloaded-artifact-from-github-releases) section.

You can now enable the Nexus Repository secrets plugin:
```sh
$ vault secrets enable -path nexus nexus
```

> [!NOTE]
> All examples in this docs assumes that the plugin is mounted at backend path `nexus`.


### Upgrade plugin version

When upgrading, please refer to the [Vault documentation](https://developer.hashicorp.com/vault/docs/upgrading/plugins) for detailed instructions.


---
## CONFIGURATION

### Nexus Repository

Create an "admin" user with a role with minimum privileges (refer to [Nexus Repository roles docs](https://help.sonatype.com/en/roles.html)):
```
nx-users-create
nx-user-delete
nx-userschangepw
```

or:
```
nx-user-all
```

### Vault secrets engine configuration

Enable the Nexus Repository secrets engine:
```sh
$ vault secrets enable -path nexus nexus
```

Config secrets engine with above prepared "admin" user:
```sh
$ vault write nexus/config/admin \
    url="https://nexus.myorg.domain" \
    username="vault-secrets-admin" \
    password="examplePassword"
```

(Optional, recommended) Rotate "admin" user's password so only Vault knows the password:
```sh
$ vault write -f nexus/config/rotate
```


---
## USAGE

### Generate dynamically Nexus Repository user with predefined roles


Create a (Vault) role:
```sh
$ vault write nexus/roles/test \
  nexus_roles="repo-a-readonly,repo-b-upload" \
  user_id_template="{{ printf \"v-%s-%s-%s\" (.RoleName) (.DisplayName | truncate 64) (unix_time) | truncate 128 | lowercase }}" \
  user_email="test@example.org" \
  ttl=10m \
  max_ttl=1h
```
```console
Success! Data written to: nexus/roles/test
```

List roles:
```sh
$ vault list nexus/roles
```
```console
Keys
----
test
```

Read role configs:
```sh
$ vault read nexus/roles/test
```
```console
Key                 Value
---                 -----
max_ttl             1h
name                test
nexus_roles         [repo-a-readonly repo-b-upload]
ttl                 10m
user_email          test@example.org
user_id_template    {{ printf "v-%s-%s-%s" (.RoleName) (.DisplayName | truncate 64) (unix_time) | truncate 128 | lowercase }}
```

Request credential for the role (password truncated):
```sh
$ vault read nexus/creds/test
```
```console
Key                Value
---                -----
lease_id           nexus/creds/test/3hwLPfGc4SQdtxUMiI0J09MQ
lease_duration     10m
lease_renewable    true
email_address      test@example.org
nexus_roles        [repo-a-readonly repo-b-upload]
password           R0QMi7RhoqqauIfERBg15Nsma2CUx9K3...
user_id            v-test-token-1733114948
```

---
## REFERENCES

### Admin Config

| Command | Path |
| ------- | ---- |
| write   | nexus/config/admin |
| read    | nexus/config/admin |
| delete  | nexus/config/admin |

Configure the parameters used to connect to the Nexus Repository server integrated with this backend.

The three main parameters: `url` which is the absolute URL to the Nexus Repository server. Note that `/v1/rest`
is prepended by the individual calls, so do not include it in the URL here.
The second and third is `username`, `password` pair which is a precreated user with enough permissions to manage other users.

An optional `insecure` parameter will enable bypassing the TLS connection verification with Nexus Repository server.

An optional `timeout` parameter is the timeout when this secrets engine calls to Nexus Repository server.

No renewals or new tokens will be issued if the backend configuration (config/admin) is deleted.

#### Parameters

* `url` (string) - Address of the Nexus Repository server instance, e.g. https://nexus.myorg.domain
* `username` (string) - The "admin" username to access Nexus Repository API.
* `password` (string) - The "admin" password.
* `insecure` (boolean) - Optional. Bypass certification verification for TLS connection with Nexus Repository API. Default to `false`.
* `timeout` (time duration) - Optional. Timeout for connection with Nexus Repository API. Default to `30s` (30 seconds).

#### Example

```sh
$ vault write nexus/config/admin \
  url="https://nexus.myorg.domain" \
  username="vault-secrets-admin" \
  password="adminPassword" \
  insecure=false \
  timeout=30s
```


### Rotate Admin Credential

| Command | Path |
| ------- | ---- |
| write   | nexus/config/rotate |

Rotate (change) the "admin" user's password used to access Nexus Repository from this plugin.

#### Examples

```sh
$ vault write -f nexus/config/rotate
```


### Role Config

| Command | Path |
| ------- | ---- |
| write   | nexus/roles/:rolename |
| patch   | nexus/roles/:rolename |
| read    | nexus/roles/:rolename |
| delete  | nexus/roles/:rolename |

Configure the parameters used to dynamically generate Nexus Repository users by the (Vault) role.

#### Parameters

* `nexus_roles` (list string) - Comma-separated string or list of predefined or precreated roles on Nexus Repository that generated users will be attatched to. Please refer to [Nexus Repository roles docs](https://help.sonatype.com/en/roles.html) for more detailed instructions.
* `user_id_template` (string) - Optional. Template for dynamically generated user ID (is username also). Default to `{{ printf "v-%s-%s-%s-%s" (.RoleName | truncate 64) (.DisplayName | truncate 64) (unix_time) (random 24) | truncate 192 | lowercase }}`. See [username templating](https://developer.hashicorp.com/vault/docs/concepts/username-templating) for details on how to write a custom template.
* `user_email` (string) - Optional. Email for generated users. Default to `no-one@example.org`.
* `ttl` (int64) - Default TTL for generated user. If unset or set to `0` uses the backend's `default_ttl`. Cannot exceed `max_ttl`.
* `max_ttl` (int64) - Maximum TTL that a credential (and generated user's lifecycle) can be renewed for. If unset or set to `0`, uses the backend's `max_ttl`. Cannot exceed backend's `max_ttl`.

#### Examples

```sh
$ vault write nexus/roles/test \
  nexus_roles="repo-a-readonly,repo-b-upload" \
  user_id_template="{{ printf \"v-%s-%s-%s\" (.RoleName) (.DisplayName | truncate 64) (unix_time) | truncate 128 | lowercase }}" \
  user_email="test@example.org" \
  ttl=10m \
  max_ttl=1h

$ vault read nexus/roles/test

$ vault delete nexus/roles/test
```


### Credential

| Command | Path |
| ------- | ---- |
| read    | nexus/creds/:rolename |

Get credential (dynamically generate Nexus Repository users) from a specified (Vault) role.

#### Responses

* `user_id` (string) - User ID of generated user.
* `email_address` (string) - Email of generated user.
* `nexus_roles` (list string) - List of roles on Nexus Repository that generated user is attatched to.
* `password` (string) - Password of generated user.

  (And [Vault's lease fields](https://developer.hashicorp.com/vault/docs/concepts/lease)):
* `lease_id` (string)
* `lease_duration` (time duration)
* `lease_renewable` (boolean)

#### Examples

```sh
$ vault read nexus/creds/test
```
```console
Key                Value
---                -----
lease_id           nexus/creds/test/b00JMCaEvlPCu5WeILCcuijR
lease_duration     10m
lease_renewable    true
email_address      test@example.org
nexus_roles        [repo-a-readonly repo-b-upload]
password           um4q4sqx5lJPpsSo8tklSKj6Ic... (password truncated)
user_id            v-test-token-1733126698
```

---
## SECURITY

Because of running on and managaging critical systems (Vault, Nexus), we all understand that this plugin can be a vulnerable part due to supply chain integrity weaknesses.
Therefore, to prove that the released binary has not been tampered with and can be securely traced back to source, the plugin is built and attested to the provenance of its release artifacts in the SLSA standard and provisionally meet Level 3 using [`SLSA` framework](https://security.googleblog.com/2021/06/introducing-slsa-end-to-end-framework.html)'s generator and Level 2 using [`GitHub's artifact attestation`](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds#about-artifact-attestations).


### Verify downloaded artifact from GitHub releases

Use either or both methods below

#### Using SLSA verifier

> [!NOTE]
> `slsa-verifier` is a tool for verifying SLSA provenance that was generated by CI/CD builders. slsa-verifier verifies the provenance by verifying the cryptographic signatures on provenance to make sure it was created by the expected builder. It then verifies that various values such as the builder id, source code repository, ref (branch or tag) matches the expected values.

Follow this [installation instruction](https://github.com/slsa-framework/slsa-verifier#download-the-binary) to install.

Verify (e.g for the `vault-plugin-secrets-nexus-repository_v1.0.0-tb1_linux-amd64.tar.gz`, version `v1.0.0-tb1`, file `vault-plugin-secrets-nexus-repository_v1.0.0-tb1.intoto.jsonl` also has to be downloaded from the release):
```sh
$ slsa-verifier verify-artifact \
  --source-uri manhtukhang/vault-plugin-secrets-nexus-repository \
  --source-tag v1.0.0-tb1 \
  --provenance-path ./vault-plugin-secrets-nexus-repository_v1.0.0-tb1.intoto.jsonl \
  ./vault-plugin-secrets-nexus-repository_v1.0.0-tb1_linux-amd64.tar.gz
```
```console
Verifying vault-plugin-secrets-nexus-repository_v0.0.1-tb1_linux-amd64.tar.gz with slsa-verifier
Verified signature against tlog entry index 155157774 at URL: https://rekor.sigstore.dev/api/v1/log/entries/108e9186e8c5677aeda7d7a4c8fb7cdb79ecaab29fe4441d8d5bd887e2e8df15d685c84350e21035
Verified build using builder "https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@refs/tags/v2.0.0" at commit 067f1fc9064d7d78d673925f78d0f81bb84c6637
Verifying artifact vault-plugin-secrets-nexus-repository_v0.0.1-tb1_linux-amd64.tar.gz: PASSED

PASSED: SLSA verification passed
```

#### Using GitHub CLI

Follow this [installation instruction](https://github.com/cli/cli#installation) to install GitHub CLI (`gh` command).

Verify (e.g for the `vault-plugin-secrets-nexus-repository_v0.0.1-tb1_linux-amd64.tar.gz`):
```sh
$ gh attestation verify \
  --repo manhtukhang/vault-plugin-secrets-nexus-repository \
  ./vault-plugin-secrets-nexus-repository_v0.0.1-tb1_linux-amd64.tar.gz
```
```console
Loaded digest sha256:815378c1be24539758452a232bb10510fb89dc562cbd1c8cd5e5746847670b58 for file://vault-plugin-secrets-nexus-repository_v0.0.1-tb1_linux-amd64.tar.gz
Loaded 1 attestation from GitHub API
✓ Verification succeeded!

sha256:815378c1be24539758452a232bb10510fb89dc562cbd1c8cd5e5746847670b58 was attested by:
REPO                                               PREDICATE_TYPE                  WORKFLOW
manhtukhang/vault-plugin-secrets-nexus-repository  https://slsa.dev/provenance/v1  .github/workflows/release.yaml@refs/tags/v0.0.1-tb1
```

> [!NOTE]
> Artifact attestations can be verified from a machine without the internet connection, please follow [this step-by-step guide](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/verifying-attestations-offline).


---
## ROADMAP

### Planned

* [ ] Support [OpenBao (Vault OSS fork)](https://openbao.org/docs/what-is-openbao) (it might already work but untested now).

### Considering

Please open issue if you have any usecase of the following features:
* [ ] Check (and ensure) if roles in `nexus_roles` existed on Nexus Repository server when the (Vault) role is create.
* [ ] Create dynamically role on Nexus Repository when (Vault) role config is created, by specified a list of Nexus privileges.
* [ ] Allow cache the previous credential (generated user) by each role (and from a same bound claim user) to avoid creating to many users with the same privileges and reduce API abusing.
* [ ] (Request your own)...
