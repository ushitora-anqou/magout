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

func expectComponentDeploy(ctx context.Context, name, namespace, mastodonServerName, image, imageBase64 string) {
	var deploy appsv1.Deployment
	err := k8sClient.Get(
		ctx,
		types.NamespacedName{Name: name, Namespace: namespace},
		&deploy,
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(deploy.Spec.Template.Spec.Containers[0].Image).To(Equal(image))
	Expect(deploy.Spec.Template.GetLabels()[labelMagoutAnqouNetMastodonServer]).To(Equal(mastodonServerName))
	Expect(deploy.Spec.Template.GetLabels()[labelMagoutAnqouNetDeployImage]).To(Equal(imageBase64))
}

func createComponentPod(ctx context.Context, name, namespace, mastodonServerName, image, imageBase64 string) {
	var pod corev1.Pod
	pod.SetName(name)
	pod.SetNamespace(namespace)
	pod.SetLabels(map[string]string{
		labelMagoutAnqouNetMastodonServer: mastodonServerName,
		labelMagoutAnqouNetDeployImage:    imageBase64,
	})
	pod.Spec.Containers = []corev1.Container{
		{Name: "container", Image: image},
	}
	err := k8sClient.Create(ctx, &pod)
	Expect(err).NotTo(HaveOccurred())
	pod.Status.Phase = corev1.PodRunning
	err = k8sClient.Status().Update(ctx, &pod)
	Expect(err).NotTo(HaveOccurred())
}

func deleteComponentPod(ctx context.Context, name, namespace string) {
	var pod corev1.Pod
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pod)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Delete(ctx, &pod)
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("MastodonServer Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		It("should run successfully", func() {
			mastodonServerName := "test"
			namespace := "default"
			namespacedName := types.NamespacedName{Name: mastodonServerName, Namespace: namespace}
			webImage := "web-image"
			webImageBase64 := "d2ViLWltYWdl"
			webImage2 := "web-image2"
			webImage2Base64 := "d2ViLWltYWdlMg"
			sidekiqImage := "sidekiq-image"
			sidekiqImageBase64 := "c2lkZWtpcS1pbWFnZQ"
			sidekiqImage2 := "sidekiq-image2"
			sidekiqImage2Base64 := "c2lkZWtpcS1pbWFnZTI"
			streamingImage := "streaming-image"
			streamingImageBase64 := "c3RyZWFtaW5nLWltYWdl"
			streamingImage2 := "streaming-image2"
			streamingImage2Base64 := "c3RyZWFtaW5nLWltYWdlMg"

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

			By("Reconciling the resource")
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

			By("Reconciling the resource")
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

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking deployments are created")
			expectComponentDeploy(ctx, mastodonServerName+"-web", namespace,
				mastodonServerName, webImage, webImageBase64)
			expectComponentDeploy(ctx, mastodonServerName+"-sidekiq", namespace,
				mastodonServerName, sidekiqImage, sidekiqImageBase64)
			expectComponentDeploy(ctx, mastodonServerName+"-streaming", namespace,
				mastodonServerName, streamingImage, streamingImageBase64)

			By("Creating pods to emulate the situation where all the deployments are ready")
			createComponentPod(ctx, mastodonServerName+"-web-ffffff", namespace,
				mastodonServerName, webImage, webImageBase64)
			createComponentPod(ctx, mastodonServerName+"-sidekiq-ffffff", namespace,
				mastodonServerName, sidekiqImage, sidekiqImageBase64)
			createComponentPod(ctx, mastodonServerName+"-streaming-ffffff", namespace,
				mastodonServerName, streamingImage, streamingImageBase64)

			By("Reconciling the resource")
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

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the MantleServer resource does NOT have migrating field")
			err = k8sClient.Get(ctx, namespacedName, server)
			Expect(err).NotTo(HaveOccurred())
			Expect(server.Status.Migrating).To(BeNil())

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Updating the MastodonServer resource")
			server.Spec.Web.Image = webImage2
			server.Spec.Sidekiq.Image = sidekiqImage2
			server.Spec.Streaming.Image = streamingImage2
			err = k8sClient.Update(ctx, server)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the MantleServer resource has migrating field")
			err = k8sClient.Get(ctx, namespacedName, server)
			Expect(err).NotTo(HaveOccurred())
			Expect(server.Status.Migrating.Web).To(Equal(webImage2))
			Expect(server.Status.Migrating.Sidekiq).To(Equal(sidekiqImage2))
			Expect(server.Status.Migrating.Streaming).To(Equal(streamingImage2))

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that a pre migration job was created")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      mastodonServerName + "-pre-migration",
				Namespace: namespace,
			}, &job)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal(webImage2))
			Expect(job.Spec.Template.Spec.Containers[0].Env).To(BeNil())

			By("Making the pre migration job completed")
			job.Status.Succeeded = 1
			err = k8sClient.Status().Update(ctx, &job)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking deployments are created")
			expectComponentDeploy(ctx, mastodonServerName+"-web", namespace,
				mastodonServerName, webImage2, webImage2Base64)
			expectComponentDeploy(ctx, mastodonServerName+"-sidekiq", namespace,
				mastodonServerName, sidekiqImage2, sidekiqImage2Base64)
			expectComponentDeploy(ctx, mastodonServerName+"-streaming", namespace,
				mastodonServerName, streamingImage2, streamingImage2Base64)

			By("Recreating the pods for the deployments")
			deleteComponentPod(ctx, mastodonServerName+"-web-ffffff", namespace)
			deleteComponentPod(ctx, mastodonServerName+"-sidekiq-ffffff", namespace)
			deleteComponentPod(ctx, mastodonServerName+"-streaming-ffffff", namespace)
			createComponentPod(ctx, mastodonServerName+"-web-ffffff", namespace,
				mastodonServerName, webImage2, webImage2Base64)
			createComponentPod(ctx, mastodonServerName+"-sidekiq-ffffff", namespace,
				mastodonServerName, sidekiqImage2, sidekiqImage2Base64)
			createComponentPod(ctx, mastodonServerName+"-streaming-ffffff", namespace,
				mastodonServerName, streamingImage2, streamingImage2Base64)

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that a post migration job was created")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      mastodonServerName + "-post-migration",
				Namespace: namespace,
			}, &job)
			Expect(err).NotTo(HaveOccurred())

			By("Making the post migration job completed")
			job.Status.Succeeded = 1
			err = k8sClient.Status().Update(ctx, &job)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the pre migration job was removed")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      mastodonServerName + "-pre-migration",
				Namespace: namespace,
			}, &job)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())

			By("Reconciling the resource")
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

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the MantleServer resource does NOT have migrating field")
			err = k8sClient.Get(ctx, namespacedName, server)
			Expect(err).NotTo(HaveOccurred())
			Expect(server.Status.Migrating).To(BeNil())

			By("Reconciling the resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
