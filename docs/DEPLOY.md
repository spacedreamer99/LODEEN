# Deploy Runbook — LODEEN prod

## Шаг 1. VPS (Terraform)

Конфиги: deploy/terraform-clo (FirstVDS) или deploy/terraform-yc (Yandex Cloud).

    cd deploy/terraform-clo
    cp terraform.tfvars.example terraform.tfvars
    # заполнить: token, ssh key
    terraform init
    terraform apply

Terraform выведет публичный IP сервера.

## Шаг 2. Bootstrap (Ansible)

    cd deploy/ansible
    cp inventory/hosts.ini.example inventory/hosts.ini
    # вписать IP сервера из terraform output
    ansible-playbook playbooks/bootstrap.yml

Что делает playbook:
- common: UFW, fail2ban, unattended-upgrades
- k3s: установка k3s без traefik + helm
- argocd: ArgoCD + Application LODEEN

## Шаг 3. ArgoCD UI

    ssh -L 8081:localhost:8081 root@IP       "k3s kubectl -n argocd port-forward svc/argocd-server 8081:443"

Пароль admin:

    ssh root@IP       "k3s kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath={.data.password} | base64 -d; echo"

Логин admin.

## Известные проблемы

- registry.terraform.io блокируется из РФ. Зеркало в ~/.terraformrc:

      provider_installation {
        network_mirror { url = "https://registry.tf-provedor.ru/" }
        direct { exclude = ["registry.terraform.io/*/*"] }
      }

- Образ LODEEN для prod: ghcr.io/spacedreamer99/lodeen-server:main
- Stateful world нельзя масштабировать в 2+ реплики (Recreate strategy)

## TODO

- cert-manager + ingress-nginx для TLS
- kube-prometheus-stack на VPS
- Velero для бэкапов etcd
- Federation между двумя кластерами
