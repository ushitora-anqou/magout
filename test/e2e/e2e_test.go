package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
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

func waitCondition(kind, name, namespace, condition string) error {
	_, _, err := kubectl(nil, "wait", "-n", namespace,
		"--for=condition="+condition, "--timeout=1s", kind, name)
	return err
}

func waitDeployAvailable(name, namespace string) error {
	return waitCondition("deployment", name, namespace, "Available")
}

func httpGet(uri string) error {
	_, _, err := kubectl(nil, "exec", "deploy/toolbox", "--",
		"curl", "--silent", uri)
	return err
}

func checkMastodonVersion(host, endpoint, expected string) error {
	stdout, _, err := kubectl(nil, "exec", "deploy/toolbox", "--",
		"curl", "--silent", "-H", "Host: "+host, endpoint+"/api/v1/instance")
	if err != nil {
		return err
	}

	result := map[string]any{}
	if err := json.Unmarshal(stdout, &result); err != nil {
		return err
	}

	version, ok := result["version"]
	if !ok {
		return errors.New("version not found in the result")
	}

	if version != expected {
		return errors.New("version not expected")
	}

	return nil
}

var _ = Describe("controller", Ordered, func() {
	Context("Operator", func() {
		It("should run successfully", func() {
			var err error

			namespace := "e2e"

			_, _, err = helm(nil, "upgrade", "--install", "--namespace", namespace,
				"magout-mastodon-server", "../../charts/magout-mastodon-server", "--wait",
				"-f", "testdata/values-v4.2.12.yaml")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() error {
				if err := waitDeployAvailable("mastodon0-web", namespace); err != nil {
					return err
				}
				if err := waitDeployAvailable("mastodon0-sidekiq", namespace); err != nil {
					return err
				}
				if err := waitDeployAvailable("mastodon0-streaming", namespace); err != nil {
					return err
				}
				if err := waitDeployAvailable("mastodon0-gateway", namespace); err != nil {
					return err
				}
				if err := httpGet("http://mastodon0-gateway.e2e.svc/health"); err != nil {
					return err
				}
				if err := checkMastodonVersion("mastodon.test",
					"http://mastodon0-gateway.e2e.svc", "4.2.12"); err != nil {
					return err
				}
				return nil
			}).Should(Succeed())
		})
	})
})
