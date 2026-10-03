# Образ Ubuntu 24.04 LTS
data "yandex_compute_image" "ubuntu" {
  family = "ubuntu-2404-lts"
}

resource "yandex_compute_instance" "lodeen" {
  name        = "lodeen-prod-1"
  platform_id = "standard-v3"
  zone        = var.yc_zone

  resources {
    cores         = 2
    memory        = 4
    core_fraction = 20 # базовая производительность ядра
  }

  boot_disk {
    initialize_params {
      image_id = data.yandex_compute_image.ubuntu.id
      size     = 40
      type     = "network-hdd"
    }
  }

  network_interface {
    subnet_id = yandex_vpc_subnet.lodeen.id
    nat       = true # публичный IP
  }

  metadata = {
    ssh-keys = "ubuntu:${file(var.ssh_public_key_path)}"
  }

  # ⬇️ ВОТ ОНО: прерываемая ВМ
  scheduling_policy {
    preemptible = true
  }
}
