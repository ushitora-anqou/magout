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
	helmPrefix    = os.Getenv("HELM")
)

func command(input []byte, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command(args[0], args[1:]...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	err := command.Run()
	if err != nil {
		slog.Error(
			"failed to execute command",
			"command", args,
			"stdout", stdout.Bytes(),
			"stderr", stderr.Bytes(),
		)
	}
	return stdout.Bytes(), stderr.Bytes(), err
}

func kubectl(input []byte, args ...string) ([]byte, []byte, error) {
	fields := strings.Fields(kubectlPrefix)
	fields = append(fields, args...)
	return command(input, fields...)
}

func kubectlApply(fileName string) error {
	_, _, err := kubectl(nil, "apply", "-f", "testdata/"+fileName)
	return err
}

func helm(input []byte, args ...string) ([]byte, []byte, error) {
	fields := strings.Fields(helmPrefix)
	fields = append(fields, args...)
	return command(input, fields...)
}

var _ = Describe("controller", Ordered, func() {
	Context("Operator", func() {
		It("should run successfully", func() {
			var err error

			_, _, err = helm(nil, "upgrade", "--install", "--namespace=e2e",
				"magout-mastodon-server", "../../charts/magout-mastodon-server", "--wait",
				"-f", "testdata/values-v4.2.12.yaml")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
