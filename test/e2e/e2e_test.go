package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func helm(input []byte, args ...string) ([]byte, []byte, error) {
	fields := strings.Fields(helmPrefix)
	fields = append(fields, args...)
	return command(input, fields...)
}

func waitNotFound(kind, name, namespace string) error {
	_, stderr, err := kubectl(nil, "get", "-n", namespace, kind, name)
	if strings.Contains(string(stderr), fmt.Sprintf("\"%s\" not found", name)) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("found")
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

func checkSchemaMigrationsCount(namespace string, expected int) error {
	stdout, _, err := kubectl(nil, "exec", "-n", namespace, "postgres-0", "--",
		"psql", "-U", "mastodon", "mastodon_production", "-c",
		"SELECT COUNT(version) FROM schema_migrations;")
	if err != nil {
		return err
	}

	result := strings.Split(string(stdout), "\n")
	if len(result) <= 2 {
		return errors.New("invalid output of psql")
	}

	count, err := strconv.Atoi(strings.TrimSpace(result[2]))
	if err != nil {
		return errors.New("failed to parse the result")
	}

	if count != expected {
		return errors.New("count not expected")
	}

	return nil
}

func getDeployment(name, namespace string) (*appsv1.Deployment, error) {
	stdout, _, err := kubectl(nil, "get", "-n", namespace, "deploy", name, "-o", "json")
	if err != nil {
		return nil, err
	}

	var deploy appsv1.Deployment
	if err := json.Unmarshal(stdout, &deploy); err != nil {
		return nil, err
	}

	return &deploy, err
}

func checkDeployResources(
	name, namespace string,
	expected corev1.ResourceRequirements,
) error {
	deploy, err := getDeployment(name, namespace)
	if err != nil {
		return err
	}

	got := deploy.Spec.Template.Spec.Containers[0].Resources
	if !got.Limits.Cpu().Equal(*expected.Limits.Cpu()) {
		return errors.New("limits cpu not equal")
	}
	if !got.Limits.Memory().Equal(*expected.Limits.Memory()) {
		return errors.New("limits cpu not equal")
	}
	if !got.Requests.Cpu().Equal(*expected.Requests.Cpu()) {
		return errors.New("limits cpu not equal")
	}
	if !got.Requests.Memory().Equal(*expected.Requests.Memory()) {
		return errors.New("limits cpu not equal")
	}

	return nil
}

func checkDeployAnnotation(name, namespace, key, expectedValue string) error {
	deploy, err := getDeployment(name, namespace)
	if err != nil {
		return err
	}

	if deploy.GetAnnotations()[key] != expectedValue {
		return errors.New("annotation not matched")
	}

	return nil
}

func checkPodAge(mastodonServerName, namespace, component string, smallerThan time.Duration) error {
	stdout, _, err := kubectl(nil, "get", "pod", "-n", namespace, "-o", "json", "-l",
		fmt.Sprintf("app.kubernetes.io/component=%s,magout.anqou.net/mastodon-server=%s",
			component, mastodonServerName))
	if err != nil {
		return err
	}

	var podList corev1.PodList
	if err := json.Unmarshal(stdout, &podList); err != nil {
		return err
	}

	for _, pod := range podList.Items {
		dur := time.Now().UTC().Sub(pod.GetCreationTimestamp().Time)
		if dur >= smallerThan {
			return errors.New("pod is too old")
		}
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

				if err := checkDeployResources("mastodon0-web", "e2e", corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("1000Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Mi"),
					},
				}); err != nil {
					return err
				}

				if err := checkDeployAnnotation("mastodon0-sidekiq", namespace,
					"test.magout.anqou.net/role", "sidekiq"); err != nil {
					return err
				}
				if err := checkDeployAnnotation("mastodon0-streaming", namespace,
					"test.magout.anqou.net/role", "streaming"); err != nil {
					return err
				}
				if err := checkDeployAnnotation("mastodon0-web", namespace,
					"test.magout.anqou.net/role", "web"); err != nil {
					return err
				}

				if err := httpGet("http://mastodon0-gateway.e2e.svc/health"); err != nil {
					return err
				}

				if err := checkMastodonVersion("mastodon.test",
					"http://mastodon0-gateway.e2e.svc", "4.2.12"); err != nil {
					return err
				}

				if err := checkSchemaMigrationsCount(namespace, 422); err != nil {
					return err
				}
				return nil
			}).Should(Succeed())

			_, _, err = helm(nil, "upgrade", "--install", "--namespace", namespace,
				"magout-mastodon-server", "../../charts/magout-mastodon-server", "--wait",
				"-f", "testdata/values-v4.3.0b1.yaml")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() error {
				return checkSchemaMigrationsCount(namespace, 460)
			}).Should(Succeed())

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

				if err := checkSchemaMigrationsCount(namespace, 468); err != nil {
					return err
				}
				return nil
			}).Should(Succeed())

			Eventually(func() error {
				return checkMastodonVersion("mastodon.test",
					"http://mastodon0-gateway.e2e.svc", "4.3.0-beta.1")
			}).Should(Succeed())

			_, _, err = helm(nil, "upgrade", "--install", "--namespace", namespace,
				"magout-mastodon-server", "../../charts/magout-mastodon-server", "--wait",
				"-f", "testdata/values-v4.3.0b1-restart.yaml")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() error {
				return checkPodAge("mastodon0", namespace, "web", 30*time.Second)
			}).Should(Succeed())
			Consistently(func() error {
				return checkPodAge("mastodon0", namespace, "web", 90*time.Second)
			}, "100s", "1s").Should(Succeed())
			Eventually(func() error {
				return checkPodAge("mastodon0", namespace, "sidekiq", 30*time.Second)
			}).Should(Succeed())
			Consistently(func() error {
				return checkPodAge("mastodon0", namespace, "sidekiq", 90*time.Second)
			}, "100s", "1s").Should(Succeed())

			_, _, err = kubectl(nil, "delete", "-n", namespace, "mastodonserver", "mastodon0")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() error {
				if err := waitNotFound("deploy", "mastodon0-sidekiq", namespace); err != nil {
					return err
				}
				if err := waitNotFound("deploy", "mastodon0-streaming", namespace); err != nil {
					return err
				}
				if err := waitNotFound("deploy", "mastodon0-web", namespace); err != nil {
					return err
				}
				return nil
			}).Should(Succeed())
		})
	})
})
