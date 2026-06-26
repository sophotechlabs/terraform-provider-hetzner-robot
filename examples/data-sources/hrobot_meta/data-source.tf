data "hrobot_meta" "current" {}

output "provider_version" {
  value = data.hrobot_meta.current.version
}
