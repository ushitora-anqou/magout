package e2e_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(200 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)

	_, _ = fmt.Fprintf(GinkgoWriter, "Starting magout suite\n")
	RunSpecs(t, "e2e suite")
}
