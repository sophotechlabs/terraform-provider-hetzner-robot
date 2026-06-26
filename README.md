# terraform-provider-hetzner-robot

[![Terraform Registry](https://img.shields.io/badge/registry-hetzner--robot-7B42BC?logo=terraform)](https://registry.terraform.io/providers/sophotechlabs/hetzner-robot/latest)
[![CI](https://github.com/sophotechlabs/terraform-provider-hetzner-robot/actions/workflows/ci.yml/badge.svg)](https://github.com/sophotechlabs/terraform-provider-hetzner-robot/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Terraform provider for the **Hetzner Robot webservice** — the dedicated-server API at `https://robot-ws.your-server.de`, *not* hcloud/cloud.

Robot manages physical dedicated servers: the switch-level firewall, rescue boot configuration, reverse DNS, and the SSH key store. This provider lets Terraform own that **declarative** Robot state with a reviewable `plan`/`apply` and drift detection, instead of shelling REST calls through ad-hoc scripts. No first-party Terraform provider exists for the Robot API.

> **Status: early development.** `v0.1.x` ships a scaffolding `hrobot_meta` data source while the release/publish pipeline is established. Real Hetzner Robot resources land in subsequent releases — see [Roadmap](#roadmap).

## Usage

```hcl
terraform {
  required_providers {
    hrobot = {
      source  = "sophotechlabs/hetzner-robot"
      version = "~> 0.1"
    }
  }
}

provider "hrobot" {}

data "hrobot_meta" "current" {}

output "provider_version" {
  value = data.hrobot_meta.current.version
}
```

The published package is `sophotechlabs/hetzner-robot`; the local provider name and all resources use the clean `hrobot` prefix (e.g. `hrobot_ssh_key`). A hyphenated source with a hyphen-free local name is a supported registry pattern (precedent: `hashicorp/google-beta`).

## Authentication

When real resources land, the provider authenticates against the Robot webservice with HTTP Basic Auth using a dedicated **webservice user** (Robot UI → Settings → "Web service and app settings" — *not* the main account login), via `HROBOT_USERNAME` / `HROBOT_PASSWORD`. The scaffolding `hrobot_meta` data source needs no credentials.

## Local development

Build the binary and point Terraform at it with a `dev_overrides` block — no `terraform init` required:

```hcl
# dev.tfrc  (pass via TF_CLI_CONFIG_FILE)
provider_installation {
  dev_overrides {
    "sophotechlabs/hetzner-robot" = "/abs/path/to/terraform-provider-hetzner-robot"
  }
  direct {}
}
```

```sh
make build          # build the provider binary
make test           # unit tests
make testacc        # acceptance tests (TF_ACC=1)
make lint           # golangci-lint
make generate       # regenerate docs/ with tfplugindocs
```

## Roadmap

| Version | Scope |
|---|---|
| `v0.1` | Scaffold + hello-world `hrobot_meta` data source + signed publish pipeline |
| `v0.2` | Robot client + Basic Auth; `hrobot_server` data source (read server facts) |
| `v0.3` | `hrobot_ssh_key` resource (`/key`, keyed by fingerprint) + import |
| `v0.4` | `hrobot_firewall` resource (switch-level, with a required return-traffic ack rule) |
| `v0.5` | `hrobot_rescue` (rescue boot config) and `hrobot_rdns` (reverse DNS) resources |
| later  | `hrobot_reset` as a Terraform action (imperative reset stays out of declarative state) |

## Non-goals

- **Server ordering / cancellation** — money-moving and irreversible; never driven from Terraform.
- **Imperative reset / installimage / OS provisioning** — an event, not desired state; handled out of band.

## License

[MIT](LICENSE) © Sophotech
