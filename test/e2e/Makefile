# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
.PHONY: test  # Run the e2e tests against a Kind k8s instance that is spun up.
test: kind helm kubectl
	$(MAKE) clean
	$(MAKE) -C ../.. docker-build IMG=$(IMG)
	$(KIND) create cluster --name $(KIND_TEST_CLUSTER) --image kindest/node:v$(KUBERNETES_VERSION)
	$(KIND) load docker-image $(IMG) --name $(KIND_TEST_CLUSTER)
	$(HELM) upgrade --create-namespace --install --namespace=magout magout ../../charts/magout --wait -f testdata/values.yaml
	$(KUBECTL) create namespace e2e
	$(KUBECTL) apply -f testdata/secret-env.yaml
	$(KUBECTL) apply -f testdata/toolbox.yaml
	$(KUBECTL) apply -f testdata/postgres.yaml
	$(KUBECTL) apply -f testdata/redis.yaml
	KUBECTL=$(KUBECTL) go test . -v -ginkgo.v

.PHONY: clean
clean: kind
	$(KIND) delete cluster --name $(KIND_TEST_CLUSTER) || true

include ../../common.mk
