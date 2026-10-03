resource "yandex_vpc_network" "lodeen" {
  name = "lodeen-network"
}

resource "yandex_vpc_subnet" "lodeen" {
  name           = "lodeen-subnet"
  zone           = var.yc_zone
  network_id     = yandex_vpc_network.lodeen.id
  v4_cidr_blocks = ["10.0.0.0/24"]
}
