.DEFAULT=local-webhook-k3d
WEBHOOK_PROXY_NAME=webhook-proxy
WEBHOOK_NAMESPACE=default
local-webhook-k3d:
	$(call check-var,IP_ADDR)

	@echo "export IP_ADDR of your computer before executing"
	./get-webhook-certs.sh
	kubectl scale deployment --namespace $(WEBHOOK_NAMESPACE) warden-webhook --replicas=0

	cat haproxy.cfg.tpl | envsubst > haproxy.cfg

	docker build -t $(WEBHOOK_PROXY_NAME) -f dockerfile.haproxy .
	$(eval HASH_TAG=$(shell docker images $(WEBHOOK_PROXY_NAME):latest --quiet))
	docker tag $(WEBHOOK_PROXY_NAME) $(WEBHOOK_PROXY_NAME):$(HASH_TAG)
	k3d image import $(WEBHOOK_PROXY_NAME):$(HASH_TAG) --cluster kyma

	cat webhook-proxy.yaml.tpl | WEBHOOK_PROXY_NAME=$(WEBHOOK_PROXY_NAME) HASH_TAG=$(HASH_TAG) envsubst | kubectl apply -f -

update-certs:
	./get-webhook-certs.sh

haproxy-config:
	$(call check-var,IP_ADDR)
	cat haproxy.cfg.tpl | envsubst > haproxy.cfg

check-var = $(if $(strip $($1)),,$(error "$1" is not defined))

undo-local-webhook-k3d:
	kubectl scale deployment --namespace $(WEBHOOK_NAMESPACE) warden-webhook --replicas=1
	kubectl delete pod -n $(WEBHOOK_NAMESPACE) webhook-proxy
