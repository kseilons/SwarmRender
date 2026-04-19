# Переменные
OPERATOR_NS = aistor
STORE_NS = object-store
LICENSE = "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCJ9.eyJhaWQiOjAsImNhcCI6MCwiaWF0IjoxLjc3NDUxMTI3Mzg0Nzc5MTkxN2U5LCJpc3MiOiJzdWJuZXRAbWluLmlvIiwibGlkIjoiYTczZjY4ZjctNWUzZS00Njg4LWEwZGYtNTE1NmM3YjBlY2M0Iiwib3JnIjoiIiwicGxhbiI6IkZSRUUiLCJzdWIiOiJzYWZvbm92YW5kcmV5MDBAbGlzdC5ydSIsInRyaWFsIjpmYWxzZX0.L3md5PZLpYrSDH8MIbEc3LSvo-4uGjpmSHsdeHJ7kdYHzLY0Lx3VkoF0rb1TcN7ZOxQnnRBHjbesiAJpUDfzeCYMVEWGzaA8sXcpwAuPyKpI1cPLKbdor-VwNnE5mudX"

.PHONY: all install-operator install-store status clean

# Установка всего сразу
all: license install-operator install-store status

license:
	kubectl create secret generic aistor-license --from-literal=license=$(LICENSE) -n $(OPERATOR_NS) --dry-run=client -o yaml | kubectl apply -f -

install-operator:
	@echo "--- Installing AIStor Operator ---"
	helm repo add minio https://helm.min.io/ || true
	helm repo update
	helm upgrade --install aistor minio/aistor-operator \
		--namespace $(OPERATOR_NS) \
		--create-namespace \
		--set license=$(LICENSE) \
		-f aistor-operator-values.yaml
	@echo "Waiting for CRDs to be ready..."
	sleep 5

install-store:
	@echo "--- Installing AIStor Object Store (Tenant) ---"
	helm upgrade --install object-store minio/aistor-objectstore \
		--namespace $(STORE_NS) \
		--create-namespace \
		-f aistor-objectstore-values.yaml

# Удаление всего
clean:
	helm uninstall object-store -n $(STORE_NS)
	helm uninstall aistor -n $(OPERATOR_NS)
	kubectl delete ns $(STORE_NS) $(OPERATOR_NS)