#!/usr/bin/env bash
set -e

ACTION=${1:-help}
MASTER_IP=${2:-192.168.0.100}
TOKEN=${3:-""}

case "$ACTION" in
  init-master)
    echo "Установка k3s Server (Master) с Flannel CNI..."
    curl -sfL https://get.k3s.io | sh -s - server --flannel-backend=vxlan --write-kubeconfig-mode 644
    MASTER_HOSTNAME=$(hostname)
    kubectl label node "$MASTER_HOSTNAME" node-role=master --overwrite
    echo "Master-узел настроен. Токен для подключения рабочей ноды:"
    sudo cat /var/lib/rancher/k3s/server/node-token
    ;;

  join-worker)
    if [ -z "$TOKEN" ]; then
      echo "Использование: $0 join-worker <MASTER_IP> <NODE_TOKEN>"
      exit 1
    fi
    echo "Подключение рабочей ноды (Worker) к мастеру $MASTER_IP..."
    curl -sfL https://get.k3s.io | K3S_URL="https://${MASTER_IP}:6443" K3S_TOKEN="${TOKEN}" sh -
    ;;

  deploy-helm)
    echo "Развертывание Helm-чарта swarmrender..."
    helm upgrade --install swarmrender ./helm/swarmrender
    kubectl rollout status deployment/postgres
    kubectl rollout status deployment/minio
    kubectl rollout status deployment/kafka
    kubectl rollout status deployment/worker-service
    echo "Распределение подов по узлам:"
    kubectl get pods -o wide
    ;;

  test-failover)
    echo "Проверка отказоустойчивости и самовосстановления подов воркера..."
    kubectl get pods -l app=worker-service -o wide
    TARGET_POD=$(kubectl get pods -l app=worker-service -o jsonpath='{.items[0].metadata.name}')
    echo "Принудительное удаление пода $TARGET_POD..."
    kubectl delete pod "$TARGET_POD" --wait=false
    sleep 2
    echo "Состояние подов после удаления:"
    kubectl get pods -l app=worker-service -o wide
    ;;

  *)
    echo "Доступные команды:"
    echo "  $0 init-master                          - Инициализация Master-узла k3s"
    echo "  $0 join-worker <MASTER_IP> <NODE_TOKEN> - Подключение рабочей ноды (Worker)"
    echo "  $0 deploy-helm                          - Установка Helm-чарта"
    echo "  $0 test-failover                        - Проверка перезапуска пода рендера"
    ;;
esac
