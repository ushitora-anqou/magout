## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
KIND ?= $(LOCALBIN)/kind
HELM ?= $(LOCALBIN)/helm
YQ ?= $(LOCALBIN)/yq
KUBECTL ?= $(LOCALBIN)/kubectl

## Versions
KUBERNETES_VERSION ?= 1.31.0
CONTROLLER_TOOLS_VERSION ?= v0.16.1
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= v1.59.1
KIND_VERSION ?= v0.24.0
HELM_VERSION ?= v3.15.4
YQ_VERSION ?= v4.44.3
KUBECTL_VERSION ?= v$(KUBERNETES_VERSION)

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = $(KUBERNETES_VERSION)

# Image URL to use all building/pushing image targets
IMG ?= magout:latest

KIND_TEST_CLUSTER ?= magout-test-cluster

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: kind
kind: $(KIND)
$(KIND): $(LOCALBIN)
	$(call go-install-tool,$(KIND),sigs.k8s.io/kind,$(KIND_VERSION))

.PHONY: helm
helm: $(HELM)
$(HELM): $(LOCALBIN)
	rm -f $(HELM)
	curl https://get.helm.sh/helm-$(HELM_VERSION)-linux-amd64.tar.gz \
		| tar xvz -C $(LOCALBIN) --strip-components 1 linux-amd64/helm
	mv $(LOCALBIN)/helm $(HELM)-$(HELM_VERSION)
	ln -sf $(HELM)-$(HELM_VERSION) $(HELM)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: yq
yq: $(YQ)
$(YQ): $(LOCALBIN)
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4,$(YQ_VERSION))

.PHONY: kubectl
kubectl: $(KUBECTL)
$(KUBECTL): $(LOCALBIN)
	rm -f $(KUBECTL)
	curl -L "https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl" > $(KUBECTL)-$(KUBECTL_VERSION)
	chmod +x $(KUBECTL)-$(KUBECTL_VERSION)
	ln -sf $(KUBECTL)-$(KUBECTL_VERSION) $(KUBECTL)
