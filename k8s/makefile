# Оркестрация Рендер-Фермы в Kubernetes (k3s + Helm)
SHELL := bash

CHART_DIR = helm/swarmrender
RELEASE_NAME = swarmrender
NAMESPACE = default

MASTER_NODE ?= master-node
WORKER_NODE ?= worker-node
MASTER_IP ?= 192.168.0.109
REGISTRY ?= $(MASTER_IP):5000

# Настройка кластера k3s
k3s-master:
	curl -sfL https://get.k3s.io | sh -s - server --write-kubeconfig-mode 644
	@sudo cat /var/lib/rancher/k3s/server/node-token

k3s-label-nodes:
	kubectl label nodes $(MASTER_NODE) node-role=master --overwrite
	kubectl label nodes $(WORKER_NODE) node-role=worker --overwrite
	kubectl get nodes --show-labels

# Запуск локального Docker Registry и публикация образов
registry:
	docker run -d -p 5000:5000 --restart=always --name registry registry:2 || true

build-images:
	docker build -t $(REGISTRY)/auth-service:latest ../services/auth-service
	docker build -t $(REGISTRY)/job-service:latest ../services/job-service
	docker build -t $(REGISTRY)/worker-service:latest ../services/worker-service
	docker build -t $(REGISTRY)/stitching-service:latest ../services/stitching-service
	docker build -t $(REGISTRY)/api-gateway:latest ../services/api-gateway
	docker build -t $(REGISTRY)/frontend:latest ../frontend

push-images:
	docker pull ghcr.io/coollabsio/minio:latest && docker tag ghcr.io/coollabsio/minio:latest $(REGISTRY)/minio:latest
	@for img in auth-service job-service worker-service stitching-service api-gateway frontend minio; do \
		echo "Pushing $(REGISTRY)/$$img:latest..."; \
		docker push $(REGISTRY)/$$img:latest; \
	done

# Управление Helm-чартом
helm-lint:
	helm lint $(CHART_DIR)

helm-template:
	helm template $(RELEASE_NAME) $(CHART_DIR)

helm-install:
	helm upgrade --install $(RELEASE_NAME) $(CHART_DIR) --namespace $(NAMESPACE) --set global.registry=$(REGISTRY)

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

.PHONY: k3s-master k3s-label-nodes registry build-images push-images helm-lint helm-template helm-install helm-uninstall verify-placement chaos-test-worker
