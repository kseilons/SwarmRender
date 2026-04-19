# Список директорий для последовательного развертывания
# Порядок важен: сначала конфиги и базы, потом очереди, потом приложения
INFRA_DIRS = config postgres kafka minio keycloak
APP_DIRS = auth-service job-service stitching-service worker-service api-gateway
DIRS = $(INFRA_DIRS) $(APP_DIRS)

PROFILE = blender

all: $(DIRS)

# 1. Настройка Minikube и окружения
setup:
	minikube delete -p $(PROFILE)
	minikube start --driver=docker --memory=8g --cpus=8 --disk-size=40g --gpus all -p $(PROFILE)

addons:
	minikube addons enable ingress -p $(PROFILE)

# Установка консоли Headlamp
headlamp:
	helm repo add headlamp https://kubernetes-sigs.github.io/headlamp/
	helm repo update
	helm install my-headlamp headlamp/headlamp --namespace kube-system --version 0.14.0
	kubectl -n kube-system create serviceaccount headlamp-admin || true
	kubectl create clusterrolebinding headlamp-admin --serviceaccount=kube-system:headlamp-admin --clusterrole=cluster-admin || true

headlamp-token:
	@kubectl create token headlamp-admin -n kube-system

# 2. Сборка образов внутри Minikube
build:
	@echo "--- Building images inside Minikube ---"
	@powershell -Command "minikube -p $(PROFILE) docker-env | Invoke-Expression; \
		docker build -t auth-service:latest ./services/auth-service; \
		docker build -t job-service:latest ./services/job-service; \
		docker build -t worker-service:latest ./services/worker-service; \
		docker build -t stitching-service:latest ./services/stitching-service; \
		docker build -t api-gateway:latest ./services/api-gateway"

# 3. Развертывание
infra: $(INFRA_DIRS)
apps: $(APP_DIRS)

$(DIRS):
	@echo "--- Deploying $@ ---"
	@if [ -f $@/makefile ]; then \
		"$(MAKE)" -C $@; \
	else \
		kubectl apply -f $@/; \
	fi

clean:
	@for dir in $(DIRS); do \
		if [ -f $$dir/makefile ]; then \
			"$(MAKE)" -C $$dir clean; \
		else \
			kubectl delete -f $$dir/ --ignore-not-found; \
		fi \
	done

.PHONY: all setup addons headlamp headlamp-token build infra apps clean $(DIRS)