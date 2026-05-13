all:
	helm upgrade --install strimzi-cluster-operator oci://quay.io/strimzi-helm/strimzi-kafka-operator \
	--version 0.45.0 \
	--namespace kafka \
	-f values.yaml

cluster:
	kubectl apply -f cluster.yaml -n kafka

clean:
	kubectl delete -f cluster.yaml -n kafka
	helm uninstall strimzi-cluster-operator -n kafka

.PHONY: all cluster clean
