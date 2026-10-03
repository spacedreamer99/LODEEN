#!/usr/bin/env bash
# Поднимает все port-forward'ы для dev-стека LODEEN.
# Использование: ./scripts/pf.sh
set -euo pipefail

pkill -f "kubectl.*port-forward" 2>/dev/null || true
sleep 2

echo "==> starting port-forwards"

kubectl -n monitoring port-forward svc/kps-grafana 3000:80 \
  > /tmp/pf-grafana.log 2>&1 &
echo "  grafana      -> http://localhost:3000"

kubectl -n monitoring port-forward svc/kps-kube-prometheus-stack-prometheus 9090:9090 \
  > /tmp/pf-prometheus.log 2>&1 &
echo "  prometheus   -> http://localhost:9090"

kubectl -n monitoring port-forward svc/kps-kube-prometheus-stack-alertmanager 9093:9093 \
  > /tmp/pf-alertmanager.log 2>&1 &
echo "  alertmanager -> http://localhost:9093"

kubectl -n argocd port-forward svc/argocd-server 8081:443 \
  > /tmp/pf-argocd.log 2>&1 &
echo "  argocd       -> https://localhost:8081"

kubectl -n lodeen port-forward svc/lodeen 7777:7777 \
  > /tmp/pf-lodeen-game.log 2>&1 &
echo "  lodeen game  -> localhost:7777"

kubectl -n lodeen port-forward svc/lodeen 19091:9091 \
  > /tmp/pf-lodeen-admin.log 2>&1 &
echo "  lodeen admin -> http://localhost:19091"

sleep 3
echo
echo "==> health checks"
curl -sS -o /dev/null -w "  prometheus  %{http_code}\n" http://localhost:9090/-/healthy || true
curl -sS -o /dev/null -w "  grafana     %{http_code}\n" http://localhost:3000/login || true
curl -sk -o /dev/null -w "  argocd      %{http_code}\n" https://localhost:8081/ || true
curl -sS -o /dev/null -w "  lodeen      %{http_code}\n" http://localhost:19091/healthz || true

echo
echo "Погасить: pkill -f 'kubectl.*port-forward'"
