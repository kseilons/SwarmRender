NS = iam

all:
	kubectl apply -f keycloak.yaml -f keycloak_ingress.yaml

secrets:
	kubectl create secret generic keycloak-kdb-user-pass --from-literal=username=admin --from-literal=password='S!B\*d$zDsb=' -n $(NS)
	kubectl create secret generic keycloak-user-pass --from-literal=username=admin --from-literal=password='S!B\*d$zDsb=' -n $(NS)

clean:
	kubectl delete -f keycloak.yaml -f keycloak_ingress.yaml
    # kubectl delete secret keycloak-kdb-user-pass keycloak-user-pass -n $(NS)

.PHONY: all secrets clean
