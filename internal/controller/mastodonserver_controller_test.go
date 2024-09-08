package controller_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	magoutv1 "github.com/ushitora-anqou/magout/api/v1"
	"github.com/ushitora-anqou/magout/internal/controller"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("MastodonServer Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		It("should create post-migration job if nothing is found", func() {
			mastodonServerName := "test"
			namespace := "default"
			namespacedName := types.NamespacedName{Name: mastodonServerName, Namespace: namespace}
			webImage := "web-image"
			webImageBase64 := "d2ViLWltYWdl"
			sidekiqImage := "sidekiq-image"
			sidekiqImageBase64 := "c2lkZWtpcS1pbWFnZQ"
			streamingImage := "streaming-image"
			streamingImageBase64 := "c3RyZWFtaW5nLWltYWdl"

			controllerReconciler := &controller.MastodonServerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Creating a MastodonServer resource")
			var err error
			server := &magoutv1.MastodonServer{}
			server.SetName(mastodonServerName)
			server.SetNamespace(namespace)
			server.Spec.Web.Image = webImage
			server.Spec.Sidekiq.Image = sidekiqImage
			server.Spec.Streaming.Image = streamingImage
			err = k8sClient.Create(ctx, server)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the MantleServer resource has migrating field")
			err = k8sClient.Get(ctx, namespacedName, server)
			Expect(err).NotTo(HaveOccurred())
			Expect(server.Status.Migrating.Web).To(Equal(webImage))
			Expect(server.Status.Migrating.Sidekiq).To(Equal(sidekiqImage))
			Expect(server.Status.Migrating.Streaming).To(Equal(streamingImage))

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that a post migration job was created")
			var job batchv1.Job
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      mastodonServerName + "-post-migration",
				Namespace: namespace,
			}, &job)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal(webImage))
			Expect(job.Spec.Template.Spec.Containers[0].Env[0].Name).To(Equal("SKIP_POST_DEPLOYMENT_MIGRATIONS"))
			Expect(job.Spec.Template.Spec.Containers[0].Env[0].Value).To(Equal("true"))

			By("Making the post migration job completed")
			job.Status.Succeeded = 1
			err = k8sClient.Status().Update(ctx, &job)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking deployments are created")
			var deployWeb appsv1.Deployment
			err = k8sClient.Get(ctx, types.NamespacedName{Name: mastodonServerName + "-web", Namespace: namespace}, &deployWeb)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployWeb.Spec.Template.GetLabels()[labelMagoutAnqouNetMastodonServer]).To(Equal(mastodonServerName))
			Expect(deployWeb.Spec.Template.GetLabels()[labelMagoutAnqouNetDeployImage]).To(Equal(webImageBase64))

			var deploySidekiq appsv1.Deployment
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{Name: mastodonServerName + "-sidekiq", Namespace: namespace},
				&deploySidekiq,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(deploySidekiq.Spec.Template.GetLabels()[labelMagoutAnqouNetMastodonServer]).To(Equal(mastodonServerName))
			Expect(deploySidekiq.Spec.Template.GetLabels()[labelMagoutAnqouNetDeployImage]).To(Equal(sidekiqImageBase64))

			var deployStreaming appsv1.Deployment
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{Name: mastodonServerName + "-streaming", Namespace: namespace},
				&deployStreaming,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployStreaming.Spec.Template.GetLabels()[labelMagoutAnqouNetMastodonServer]).To(Equal(mastodonServerName))
			Expect(deployStreaming.Spec.Template.GetLabels()[labelMagoutAnqouNetDeployImage]).To(Equal(streamingImageBase64))

			By("Creating pods to emulate the situation where all the deployments are ready")
			var podWeb corev1.Pod
			podWeb.SetName(mastodonServerName + "-web-ffffff")
			podWeb.SetNamespace(namespace)
			podWeb.SetLabels(map[string]string{
				labelMagoutAnqouNetMastodonServer: mastodonServerName,
				labelMagoutAnqouNetDeployImage:    webImageBase64,
			})
			podWeb.Spec.Containers = []corev1.Container{
				{Name: "container", Image: webImage},
			}
			podWeb.Status.Phase = corev1.PodRunning
			err = k8sClient.Create(ctx, &podWeb)
			Expect(err).NotTo(HaveOccurred())

			var podSidekiq corev1.Pod
			podSidekiq.SetName(mastodonServerName + "-sidekiq-ffffff")
			podSidekiq.SetNamespace(namespace)
			podSidekiq.SetLabels(map[string]string{
				labelMagoutAnqouNetMastodonServer: mastodonServerName,
				labelMagoutAnqouNetDeployImage:    sidekiqImageBase64,
			})
			podSidekiq.Spec.Containers = []corev1.Container{
				{Name: "container", Image: sidekiqImage},
			}
			podSidekiq.Status.Phase = corev1.PodRunning
			err = k8sClient.Create(ctx, &podSidekiq)
			Expect(err).NotTo(HaveOccurred())

			var podStreming corev1.Pod
			podStreming.SetName(mastodonServerName + "-streaming-ffffff")
			podStreming.SetNamespace(namespace)
			podStreming.SetLabels(map[string]string{
				labelMagoutAnqouNetMastodonServer: mastodonServerName,
				labelMagoutAnqouNetDeployImage:    streamingImageBase64,
			})
			podStreming.Spec.Containers = []corev1.Container{
				{Name: "container", Image: streamingImage},
			}
			podStreming.Status.Phase = corev1.PodRunning
			err = k8sClient.Create(ctx, &podStreming)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the post migration job was removed")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      mastodonServerName + "-post-migration",
				Namespace: namespace,
			}, &job)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
