package e2e_test

import (
	"bytes"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	kubectlPrefix = os.Getenv("KUBECTL")
)

func kubectl(input []byte, args ...string) ([]byte, []byte, error) {
	fields := strings.Fields(kubectlPrefix)
	fields = append(fields, args...)

	var stdout, stderr bytes.Buffer
	command := exec.Command(fields[0], fields[1:]...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func kubectlApply(fileName string) error {
	stdout, stderr, err := kubectl(nil, "apply", "-f", "testdata/"+fileName)
	if err != nil {
		slog.Error("failed to kubectl apply", "stdout", stdout, "stderr", stderr)
	}
	return err
}

var _ = Describe("controller", Ordered, func() {
	Context("Operator", func() {
		It("should run successfully", func() {
			var err error

			err = kubectlApply("mastodon0-v4.2.12.yaml")
			Expect(err).NotTo(HaveOccurred())

			return
		})
	})
})
