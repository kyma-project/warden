.DEFAULT=local-webhook-k3d
ADMISSION_PROXY_NAME=admission-proxy
WEBHOOK_NAMESPACE=default
local-webhook-k3d:
	$(call check-var,IP_ADDR)

	@echo "export IP_ADDR of your computer before executing"
	./get-webhook-certs.sh
	kubectl scale deployment --namespace $(WEBHOOK_NAMESPACE) warden-webhook --replicas=0

	cat haproxy.cfg.tpl | envsubst > haproxy.cfg

	docker build -t $(ADMISSION_PROXY_NAME) -f dockerfile.haproxy .
	$(eval HASH_TAG=$(shell docker images $(ADMISSION_PROXY_NAME):latest --quiet))
	docker tag $(ADMISSION_PROXY_NAME) $(ADMISSION_PROXY_NAME):$(HASH_TAG)
	k3d image import $(ADMISSION_PROXY_NAME):$(HASH_TAG) --cluster kyma

	cat webhook-proxy.yaml.tpl | WEBHOOK_PROXY_NAME=$(ADMISSION_PROXY_NAME) HASH_TAG=$(HASH_TAG) envsubst | kubectl apply -f -

update-certs:
	./get-webhook-certs.sh

haproxy-config:
	$(call check-var,IP_ADDR)
	cat haproxy.cfg.tpl | envsubst > haproxy.cfg

check-var = $(if $(strip $($1)),,$(error "$1" is not defined))

undo-local-webhook-k3d:
	kubectl scale deployment --namespace $(WEBHOOK_NAMESPACE) warden-webhook --replicas=1
	kubectl delete pod -n $(WEBHOOK_NAMESPACE) webhook-proxy
