include ../../common.mk

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
.PHONY: test  # Run the e2e tests against a Kind k8s instance that is spun up.
test: kind helm
	$(MAKE) clean
	$(MAKE) -C ../.. docker-build IMG=$(IMG)
	$(KIND) create cluster --name $(KIND_TEST_CLUSTER) --image kindest/node:v$(KUBERNETES_VERSION)
	$(KIND) load docker-image $(IMG) --name $(KIND_TEST_CLUSTER)
	$(HELM) upgrade --create-namespace --install --namespace=e2e magout ../../charts/magout --wait -f testdata/values.yaml
	go test . -v -ginkgo.v

.PHONY: clean
clean: kind
	$(KIND) delete cluster --name $(KIND_TEST_CLUSTER) || true
