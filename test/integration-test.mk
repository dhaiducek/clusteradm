# Copyright Contributors to the Open Cluster Management project

export KUBEBUILDER_ASSETS ?=$(LOCAL_BIN)/kubebuilder/bin
export GINKGO ?=$(LOCAL_BIN)/ginkgo
.PHONY: envtest-setup

ensure-ginkgo:
	# Downloading ginkgo into '$(GINKGO)'
	GOBIN=$(LOCAL_BIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print $$2}' go.mod)
.PHONY: ensure-ginkgo

clean-integration-test:
	$(RM) '$(KB_TOOLS_ARCHIVE_PATH)'
	rm -rf $(TEST_TMP)/kubebuilder
	$(RM) ./integration.test
.PHONY: clean-integration-test

clean: clean-integration-test

test-integration: envtest-setup ensure-ginkgo
	$(GINKGO) -v ./pkg/cmd/addon/enable  ./pkg/cmd/addon/disable  ./pkg/cmd/install/hubaddon 
.PHONY: test-integration
