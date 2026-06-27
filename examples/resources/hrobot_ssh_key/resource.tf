resource "hrobot_ssh_key" "lab" {
  name       = "k3s-lab"
  public_key = file("~/.ssh/id_ed25519.pub")
}

output "lab_key_fingerprint" {
  value = hrobot_ssh_key.lab.fingerprint
}
