# Пример ресурса сервера CLO.
# Точные имена ресурсов и параметры смотри в документации провайдера:
# https://registry.terraform.io/providers/clo-ru/clo/latest/docs
# или в примерах на GitHub: https://github.com/clo-ru/terraform-provider-clo/tree/main/examples

resource "clo_server" "lodeen" {
  name = var.server_name
  plan = var.server_plan

  # SSH-ключ добавим через provisioner или cloud-init (см. ниже)
}
