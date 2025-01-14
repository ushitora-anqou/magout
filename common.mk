## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin

## Versions
KUBERNETES_VERSION ?= 1.32.0
CONTROLLER_TOOLS_VERSION ?= v0.17.1
ENVTEST_VERSION ?= v0.19.4
GOLANGCI_LINT_VERSION ?= 1.63.4
KIND_VERSION ?= v0.26.0
HELM_VERSION ?= v3.16.4
YQ_VERSION ?= v4.45.1
KUBECTL_VERSION ?= v$(KUBERNETES_VERSION)
ENVTEST_K8S_VERSION = $(KUBERNETES_VERSION)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
KIND ?= $(LOCALBIN)/kind-$(KIND_VERSION)
HELM ?= $(LOCALBIN)/helm-$(HELM_VERSION)
YQ ?= $(LOCALBIN)/yq-$(YQ_VERSION)
KUBECTL ?= $(LOCALBIN)/kubectl-$(KUBECTL_VERSION)

# Image URL to use all building/pushing image targets
IMG ?= magout:latest

KIND_TEST_CLUSTER ?= magout-test-cluster

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: kind
kind: $(KIND)
$(KIND): | $(LOCALBIN)
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/$(KIND_VERSION)/kind-linux-amd64 > $(KIND)
	chmod +x $(KIND)

.PHONY: helm
helm: $(HELM)
$(HELM): | $(LOCALBIN)
	rm -f $(HELM)
	curl https://get.helm.sh/helm-$(HELM_VERSION)-linux-amd64.tar.gz \
		| tar xvz -C $(LOCALBIN) --strip-components 1 linux-amd64/helm
	mv $(LOCALBIN)/helm $(HELM)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): | $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
	mv $(LOCALBIN)/controller-gen $(CONTROLLER_GEN)

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): | $(LOCALBIN)
	curl -L https://github.com/kubernetes-sigs/controller-runtime/releases/download/$(ENVTEST_VERSION)/setup-envtest-linux-amd64 > $(ENVTEST)
	chmod +x $(ENVTEST)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): | $(LOCALBIN)
	rm -f $(GOLANGCI_LINT)
	curl -L https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-linux-amd64.tar.gz \
		| tar xvz -C $(LOCALBIN) --strip-components 1 golangci-lint-$(GOLANGCI_LINT_VERSION)-linux-amd64/golangci-lint
	mv $(LOCALBIN)/golangci-lint $(GOLANGCI_LINT)

.PHONY: yq
yq: | $(YQ)
$(YQ): | $(LOCALBIN)
	curl -L https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_linux_amd64 > $(YQ)
	chmod +x $(YQ)

.PHONY: kubectl
kubectl: | $(KUBECTL)
$(KUBECTL): | $(LOCALBIN)
	curl -L "https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl" > $(KUBECTL)
	chmod +x $(KUBECTL)
