package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	magoutv1 "github.com/ushitora-anqou/magout/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	domain              = "magout.anqou.net"
	labelMastodonServer = domain + "/mastodon-server"
	labelDeployImage    = domain + "/deploy-image"

	componentWeb       = "web"
	componentSidekiq   = "sidekiq"
	componentStreaming = "streaming"

	jobPreMigration  = "pre-migration"
	jobPostMigration = "post-migration"

	deployStatusUnknown = iota
	deployStatusNotFound
	deployStatusReady
	deployStatusNotReady

	jobStatusUnknown = iota
	jobStatusNotFound
	jobStatusCompleted
	jobStatusNotCompleted
	jobStatusFailed
)

func buildDeploymentName(component, mastodonServerName string) string {
	return fmt.Sprintf("%s-%s", mastodonServerName, component)
}

func buildJobName(kind, mastodonServerName string) string {
	return fmt.Sprintf("%s-%s", mastodonServerName, kind)
}

func encodeDeploymentImage(image string) string {
	return base64.StdEncoding.EncodeToString([]byte(image))
}

func getDeploymentImage(
	ctx context.Context, client client.Client, name, namespace string,
) (string, error) {
	var deploy appsv1.Deployment
	if err := client.Get(
		ctx, types.NamespacedName{
			Name: buildDeploymentName(componentWeb, name), Namespace: namespace,
		}, &deploy,
	); err != nil {
		return "", err
	}

	spec := deploy.Spec.Template.Spec
	if len(spec.InitContainers) == 0 {
		return spec.Containers[0].Image, nil
	}
	return spec.InitContainers[0].Image, nil
}

type ImageMap struct {
	web, sidekiq, streaming string
}

func (im1 *ImageMap) Equals(im2 *ImageMap) bool {
	return im1.web == im2.web && im1.sidekiq == im2.sidekiq && im1.streaming == im2.streaming
}

// MastodonServerReconciler reconciles a MastodonServer object
type MastodonServerReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=magout.anqou.net,resources=mastodonservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=magout.anqou.net,resources=mastodonservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=magout.anqou.net,resources=mastodonservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MastodonServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *MastodonServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var server magoutv1.MastodonServer
	if err := r.Client.Get(ctx, req.NamespacedName, &server); k8serrors.IsNotFound(err) {
		logger.Info(
			"MastodonServer not found",
			"name", server.GetName(),
			"namespace", server.GetNamespace(),
		)
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	specImageMap := ImageMap{
		web:       server.Spec.Web.Image,
		sidekiq:   server.Spec.Sidekiq.Image,
		streaming: server.Spec.Streaming.Image,
	}

	var migratingImageMap *ImageMap
	if server.Status.Migrating != nil {
		migratingImageMap = &ImageMap{
			web:       server.Status.Migrating.Web,
			sidekiq:   server.Status.Migrating.Sidekiq,
			streaming: server.Status.Migrating.Streaming,
		}
	}

	deploymentsStatus, deployImageMap, err := r.getDeploymentsStatus(
		ctx, server.GetName(), server.GetNamespace())
	if err != nil {
		return ctrl.Result{}, err
	}

	preMigrationJobStatus, err := r.getJobStatus(
		ctx, server.GetName(), server.GetNamespace(), jobPreMigration)
	if err != nil {
		return ctrl.Result{}, err
	}

	postMigrationJobStatus, err := r.getJobStatus(
		ctx, server.GetName(), server.GetNamespace(), jobPostMigration)
	if err != nil {
		return ctrl.Result{}, err
	}

	switch {
	case
		deploymentsStatus == deployStatusNotFound &&
			preMigrationJobStatus == jobStatusNotFound &&
			postMigrationJobStatus == jobStatusNotFound &&
			migratingImageMap != nil: // S1
	default:
		err := errors.New("unknown current status")
		logger.Error(
			err,
			"manual intervention should be performed because reconciler can't handle the current situation",
			"deploymentStatus", deploymentsStatus,
			"preMigrationJobStatus", preMigrationJobStatus,
			"postMigrationJobStatus", postMigrationJobStatus,
			"migratingImageMap", migratingImageMap != nil,
		)
		return ctrl.Result{}, err
	}

	if err := r.createOrUpdateSidekiqDeployment(ctx, &server); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.createOrUpdateStreamingDeployment(ctx, &server); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.createOrUpdateWebDeployment(ctx, &server); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MastodonServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&magoutv1.MastodonServer{}).
		Complete(r)
}

func (r *MastodonServerReconciler) createOrUpdateSidekiqDeployment(
	ctx context.Context,
	server *magoutv1.MastodonServer,
) error {
	spec := server.Spec.Sidekiq
	return r.createOrUpdateDeployment(
		ctx,
		server,
		"sidekiq",
		componentSidekiq,
		spec.Annotations,
		spec.Labels,
		spec.Replicas,
		spec.Image,
		spec.Resources,
		[]string{"bash", "-c", "bundle exec sidekiq"},
		spec.EnvFrom,
		nil,
		nil,
		nil,
		nil,
	)
}

func (r *MastodonServerReconciler) createOrUpdateStreamingDeployment(
	ctx context.Context,
	server *magoutv1.MastodonServer,
) error {
	spec := server.Spec.Streaming
	return r.createOrUpdateDeployment(
		ctx,
		server,
		"node",
		componentStreaming,
		spec.Annotations,
		spec.Labels,
		spec.Replicas,
		spec.Image,
		spec.Resources,
		[]string{"bash", "-c", "node ./streaming"},
		spec.EnvFrom,
		[]corev1.ContainerPort{
			{Name: "streaming", ContainerPort: 4000, Protocol: "TCP"},
		},
		&corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromString("streaming"),
					Path: "/api/v1/streaming/health",
				},
			},
		},
		nil,
		nil,
	)
}

func (r *MastodonServerReconciler) createOrUpdateWebDeployment(
	ctx context.Context,
	server *magoutv1.MastodonServer,
) error {
	spec := server.Spec.Web
	return r.createOrUpdateDeployment(
		ctx,
		server,
		"puma",
		componentWeb,
		spec.Annotations,
		spec.Labels,
		spec.Replicas,
		spec.Image,
		spec.Resources,
		[]string{"bash", "-c", "bundle exec puma -C config/puma.rb"},
		spec.EnvFrom,
		[]corev1.ContainerPort{
			{Name: "http", ContainerPort: 3000, Protocol: "TCP"},
		},
		&corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromString("http"),
				},
			},
		},
		&corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromString("http"),
					Path: "/health",
				},
			},
		},
		&corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromString("http"),
					Path: "/health",
				},
			},
			FailureThreshold: 30,
			PeriodSeconds:    5,
		},
	)
}

func (r *MastodonServerReconciler) createOrUpdateDeployment(
	ctx context.Context,
	server *magoutv1.MastodonServer,
	appName, component string,
	deployAnnotations map[string]string,
	deployLabels map[string]string,
	replicas int32,
	image string,
	resources corev1.ResourceRequirements,
	command []string,
	envFrom []corev1.EnvFromSource,
	ports []corev1.ContainerPort,
	livenessProbe *corev1.Probe,
	readinessProbe *corev1.Probe,
	startupProbe *corev1.Probe,
) error {
	logger := log.FromContext(ctx)

	var deploy appsv1.Deployment
	deploy.SetName(buildDeploymentName(component, server.GetName()))
	deploy.SetNamespace(server.GetNamespace())

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		deploy.SetAnnotations(deployAnnotations)

		selector := map[string]string{
			"app.kubernetes.io/name":      appName,
			"app.kubernetes.io/component": component,
			"app.kubernetes.io/part-of":   "mastodon",
		}
		deploy.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: selector,
		}

		labels := map[string]string{
			labelMastodonServer: server.GetName(),
			labelDeployImage:    encodeDeploymentImage(image),
		}
		for k, v := range selector {
			labels[k] = v
		}
		for k, v := range deployLabels {
			labels[k] = v
		}
		deploy.SetLabels(labels)

		deploy.Spec.Replicas = &replicas
		deploy.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:           component,
				Image:          image,
				Resources:      resources,
				Command:        command,
				EnvFrom:        envFrom,
				Ports:          ports,
				LivenessProbe:  livenessProbe,
				ReadinessProbe: readinessProbe,
				StartupProbe:   startupProbe,
			},
		}

		return ctrl.SetControllerReference(server, &deploy, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf(
			"failed to create or update deployment: %s: %s: %w",
			deploy.GetName(),
			deploy.GetNamespace(),
			err,
		)
	}
	if result != controllerutil.OperationResultNone {
		logger.Info(
			"create deployment successfully",
			"name", deploy.GetName(),
			"namespace", deploy.GetNamespace(),
		)
	}

	return nil
}

func (r *MastodonServerReconciler) getJobStatus(
	ctx context.Context, name, namespace, kind string,
) (int, error) {
	var job batchv1.Job
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      buildJobName(name, kind),
		Namespace: namespace,
	}, &job); k8serrors.IsNotFound(err) {
		return jobStatusNotFound, nil
	} else if err != nil {
		return jobStatusUnknown, err
	}

	if job.Status.Succeeded == 1 {
		return jobStatusCompleted, nil
	}
	if job.Status.Failed == 1 {
		return jobStatusFailed, nil
	}
	return jobStatusNotCompleted, nil
}

func (r *MastodonServerReconciler) getDeploymentsStatus(
	ctx context.Context, name, namespace string,
) (int, *ImageMap, error) {
	webImage, err := getDeploymentImage(
		ctx, r.Client, buildDeploymentName(componentWeb, name), namespace)
	if k8serrors.IsNotFound(err) {
		return deployStatusNotFound, nil, nil
	} else if err != nil {
		return deployStatusUnknown, nil, err
	}

	sidekiqImage, err := getDeploymentImage(
		ctx, r.Client, buildDeploymentName(componentSidekiq, name), namespace)
	if k8serrors.IsNotFound(err) {
		return deployStatusNotFound, nil, nil
	} else if err != nil {
		return deployStatusUnknown, nil, err
	}

	streamingImage, err := getDeploymentImage(
		ctx, r.Client, buildDeploymentName(componentStreaming, name), namespace)
	if k8serrors.IsNotFound(err) {
		return deployStatusNotFound, nil, nil
	} else if err != nil {
		return deployStatusUnknown, nil, err
	}

	req1, err := labels.NewRequirement(labelMastodonServer, selection.Equals, []string{name})
	if err != nil {
		return deployStatusUnknown, nil, err
	}
	req2, err := labels.NewRequirement(labelDeployImage, selection.Exists, []string{})
	if err != nil {
		return deployStatusUnknown, nil, err
	}
	var allPods corev1.PodList
	if err := r.Client.List(ctx, &allPods, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.ValidatedSetSelector{}.Add(*req1, *req2),
	}); err != nil {
		return deployStatusUnknown, nil, err
	}

	req3, err := labels.NewRequirement(labelDeployImage, selection.In, []string{
		encodeDeploymentImage(webImage),
		encodeDeploymentImage(sidekiqImage),
		encodeDeploymentImage(streamingImage),
	})
	if err != nil {
		return deployStatusUnknown, nil, err
	}
	var livePodsList corev1.PodList
	if err := r.Client.List(ctx, &livePodsList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.ValidatedSetSelector{}.Add(*req1, *req3),
	}); err != nil {
		return deployStatusUnknown, nil, err
	}
	livePods := []corev1.Pod{}
	for _, pod := range livePodsList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		livePods = append(livePods, pod)
	}

	imageMap := ImageMap{web: webImage, sidekiq: sidekiqImage, streaming: streamingImage}

	if len(allPods.Items) == len(livePods) {
		return deployStatusReady, &imageMap, nil
	}
	return deployStatusNotReady, &imageMap, nil
}
