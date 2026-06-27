data "hrobot_server" "by_number" {
  number = 2893
}

data "hrobot_server" "by_ip" {
  ip = "176.9.18.203"
}

output "server_product" {
  value = data.hrobot_server.by_number.product
}

output "server_number_from_ip" {
  value = data.hrobot_server.by_ip.number
}
