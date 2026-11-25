# Оркестрация Рендер-Фермы в Kubernetes (k3s + Helm)
SHELL := bash

CHART_DIR = helm/swarmrender
RELEASE_NAME = swarmrender
NAMESPACE = default

MASTER_NODE ?= master-node
WORKER_NODE ?= worker-node

# Настройка кластера k3s
k3s-master:
	curl -sfL https://get.k3s.io | sh -s - server --write-kubeconfig-mode 644
	@sudo cat /var/lib/rancher/k3s/server/node-token

k3s-label-nodes:
	kubectl label nodes $(MASTER_NODE) node-role=master --overwrite
	kubectl label nodes $(WORKER_NODE) node-role=worker --overwrite
	kubectl get nodes --show-labels

# Сборка и импорт Docker-образов
build-images:
	docker build -t auth-service:latest ../services/auth-service
	docker build -t job-service:latest ../services/job-service
	docker build -t worker-service:latest ../services/worker-service
	docker build -t stitching-service:latest ../services/stitching-service
	docker build -t api-gateway:latest ../services/api-gateway
	docker build -t frontend:latest ../frontend

import-k3s:
	@for img in auth-service job-service worker-service stitching-service api-gateway frontend; do \
		echo "Importing $$img:latest into k3s containerd..."; \
		docker save $$img:latest | sudo k3s ctr images import -; \
	done

# Управление Helm-чартом
helm-lint:
	helm lint $(CHART_DIR)

helm-template:
	helm template $(RELEASE_NAME) $(CHART_DIR)

helm-install:
	helm upgrade --install $(RELEASE_NAME) $(CHART_DIR) --namespace $(NAMESPACE)

helm-uninstall:
	helm uninstall $(RELEASE_NAME) --namespace $(NAMESPACE)

# Проверка распределения подов и отказоустойчивости
verify-placement:
	kubectl get pods -o wide

chaos-test-worker:
	@POD=$$(kubectl get pods -l app=worker-service -o jsonpath='{.items[0].metadata.name}'); \
	echo "Deleting pod: $$POD"; \
	kubectl delete pod $$POD; \
	kubectl get pods -l app=worker-service -o wide -w

.PHONY: k3s-master k3s-label-nodes build-images import-k3s helm-lint helm-template helm-install helm-uninstall verify-placement chaos-test-worker
