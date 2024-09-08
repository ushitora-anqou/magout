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

// +kubebuilder:rbac:groups=magout.anqou.net,resources=mastodonservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=magout.anqou.net,resources=mastodonservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=magout.anqou.net,resources=mastodonservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

//go:generate go run golang.org/x/tools/cmd/stringer -type=deployStatusType
//go:generate go run golang.org/x/tools/cmd/stringer -type=jobStatusType
//go:generate go run golang.org/x/tools/cmd/stringer -type=whatToDoType

type (
	componentType    string
	jobType          string
	deployStatusType int
	jobStatusType    int
	whatToDoType     int
)

const (
	domain              = "magout.anqou.net"
	labelMastodonServer = domain + "/mastodon-server"
	labelDeployImage    = domain + "/deploy-image"

	componentWeb       componentType = "web"
	componentSidekiq   componentType = "sidekiq"
	componentStreaming componentType = "streaming"

	jobPreMigration  jobType = "pre-migration"
	jobPostMigration jobType = "post-migration"

	deployStatusUnknown deployStatusType = iota
	deployStatusNotFound
	deployStatusReady
	deployStatusNotReady

	jobStatusUnknown jobStatusType = iota
	jobStatusNotFound
	jobStatusCompleted
	jobStatusNotCompleted
	jobStatusFailed

	shouldCreatePreMigrationJob whatToDoType = iota
	shouldCreatePostMigrationJob
	shouldSetMigratingStatus
	shouldUnsetMigratingStatus
	shouldCreateOrUpdateDeploysWithSpec
	shouldCreateOrUpdateDeploysWithMigratingImages
	shouldDeletePostMigrationJob
	shouldDeletePreMigrationJob
	shouldDoNothing
)

func buildDeploymentName(component componentType, mastodonServerName string) string {
	return fmt.Sprintf("%s-%s", mastodonServerName, string(component))
}

func buildJobName(kind jobType, mastodonServerName string) string {
	return fmt.Sprintf("%s-%s", mastodonServerName, string(kind))
}

func encodeDeploymentImage(image string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(image))
}

func getDeploymentImage(
	ctx context.Context, cli client.Client, name, namespace string,
) (string, error) {
	var deploy appsv1.Deployment
	if err := cli.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &deploy); err != nil {
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

type k8sStatus struct {
	deploymentsStatus      deployStatusType
	preMigrationJobStatus  jobStatusType
	postMigrationJobStatus jobStatusType
	migratingImageMap      *ImageMap
	currentImageMap        *ImageMap
	specImageMap           *ImageMap
}

// MastodonServerReconciler reconciles a MastodonServer object.
type MastodonServerReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *MastodonServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&magoutv1.MastodonServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

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

	k8sStatus, err := r.fetchK8sStatus(ctx, &server)
	if err != nil {
		return ctrl.Result{}, err
	}

	whattodo, err := decideWhatToDo(k8sStatus)
	if err != nil {
		logger.Error(
			err,
			"manual intervention should be performed because reconciler can't handle the current situation",
			"deploymentStatus", k8sStatus.deploymentsStatus,
			"preMigrationJobStatus", k8sStatus.preMigrationJobStatus,
			"postMigrationJobStatus", k8sStatus.postMigrationJobStatus,
			"migratingImageMap", k8sStatus.migratingImageMap != nil,
		)
		return ctrl.Result{}, err
	}

	logger.Info(
		"reconciling",
		"name", server.GetName(),
		"namespace", server.GetNamespace(),
		"whattodo", whattodo.String(),
		"deploymentStatus", k8sStatus.deploymentsStatus,
		"preMigrationJobStatus", k8sStatus.preMigrationJobStatus,
		"postMigrationJobStatus", k8sStatus.postMigrationJobStatus,
		"migratingImageMap", k8sStatus.migratingImageMap != nil,
	)

	switch whattodo {
	case shouldCreatePreMigrationJob:
		if err := r.createMigrationJob(ctx, &server, k8sStatus.migratingImageMap, jobPreMigration); err != nil {
			return ctrl.Result{}, err
		}

	case shouldCreatePostMigrationJob:
		if err := r.createMigrationJob(ctx, &server, k8sStatus.migratingImageMap, jobPostMigration); err != nil {
			return ctrl.Result{}, err
		}

	case shouldSetMigratingStatus:
		server.Status.Migrating = &magoutv1.MastodonServerMigratingStatus{
			Web:       k8sStatus.specImageMap.web,
			Sidekiq:   k8sStatus.specImageMap.sidekiq,
			Streaming: k8sStatus.specImageMap.streaming,
		}
		if err := r.Client.Status().Update(ctx, &server); err != nil {
			return ctrl.Result{}, err
		}

	case shouldUnsetMigratingStatus:
		server.Status.Migrating = nil
		if err := r.Client.Status().Update(ctx, &server); err != nil {
			return ctrl.Result{}, err
		}

	case shouldCreateOrUpdateDeploysWithSpec:
		if err := r.createOrUpdateDeployments(ctx, &server, k8sStatus.specImageMap); err != nil {
			return ctrl.Result{}, err
		}

	case shouldCreateOrUpdateDeploysWithMigratingImages:
		if err := r.createOrUpdateDeployments(ctx, &server, k8sStatus.migratingImageMap); err != nil {
			return ctrl.Result{}, err
		}

	case shouldDeletePostMigrationJob:
		if err := r.deleteJob(
			ctx,
			buildJobName(jobPostMigration, server.GetName()),
			server.GetNamespace(),
		); err != nil {
			return ctrl.Result{}, err
		}

	case shouldDeletePreMigrationJob:
		if err := r.deleteJob(
			ctx,
			buildJobName(jobPreMigration, server.GetName()),
			server.GetNamespace(),
		); err != nil {
			return ctrl.Result{}, err
		}

	case shouldDoNothing:

	default:
		panic("unreachable")
	}

	return ctrl.Result{}, nil
}

func (r *MastodonServerReconciler) fetchK8sStatus(
	ctx context.Context,
	server *magoutv1.MastodonServer,
) (*k8sStatus, error) {
	var err error
	res := &k8sStatus{}

	res.specImageMap = &ImageMap{
		web:       server.Spec.Web.Image,
		sidekiq:   server.Spec.Sidekiq.Image,
		streaming: server.Spec.Streaming.Image,
	}

	if server.Status.Migrating != nil {
		res.migratingImageMap = &ImageMap{
			web:       server.Status.Migrating.Web,
			sidekiq:   server.Status.Migrating.Sidekiq,
			streaming: server.Status.Migrating.Streaming,
		}
	}

	res.deploymentsStatus, res.currentImageMap, err = r.getDeploymentsStatus(
		ctx, server.GetName(), server.GetNamespace())
	if err != nil {
		return nil, err
	}

	res.preMigrationJobStatus, err = r.getJobStatus(
		ctx, server.GetName(), server.GetNamespace(), jobPreMigration)
	if err != nil {
		return nil, err
	}

	res.postMigrationJobStatus, err = r.getJobStatus(
		ctx, server.GetName(), server.GetNamespace(), jobPostMigration)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *MastodonServerReconciler) createMigrationJob(
	ctx context.Context,
	server *magoutv1.MastodonServer,
	imageMap *ImageMap,
	kind jobType,
) error {
	env := []corev1.EnvVar{}
	switch kind {
	case jobPreMigration:
	case jobPostMigration:
		env = append(env, corev1.EnvVar{
			Name: "SKIP_POST_DEPLOYMENT_MIGRATIONS", Value: "true",
		})
	default:
		return errors.New("invalid job kind")
	}

	var job batchv1.Job
	job.SetName(buildJobName(kind, server.GetName()))
	job.SetNamespace(server.GetNamespace())
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
	job.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:    "migration",
			Image:   imageMap.web,
			EnvFrom: server.Spec.Web.EnvFrom,
			Env:     env,
			Command: []string{
				"bash",
				"-c",
				"bundle exec rake db:create;\nbundle exec rake db:migrate",
			},
		},
	}

	if err := ctrl.SetControllerReference(server, &job, r.Scheme); err != nil {
		return err
	}

	return r.Client.Create(ctx, &job)
}

func (r *MastodonServerReconciler) createOrUpdateDeployments(
	ctx context.Context,
	server *magoutv1.MastodonServer,
	imageMap *ImageMap,
) error {
	if err := r.createOrUpdateSidekiqDeployment(ctx, server, imageMap.sidekiq); err != nil {
		return err
	}
	if err := r.createOrUpdateStreamingDeployment(ctx, server, imageMap.streaming); err != nil {
		return err
	}
	if err := r.createOrUpdateWebDeployment(ctx, server, imageMap.web); err != nil {
		return err
	}
	return nil
}

func (r *MastodonServerReconciler) createOrUpdateSidekiqDeployment(
	ctx context.Context,
	server *magoutv1.MastodonServer,
	image string,
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
		image,
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
	image string,
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
		image,
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
	image string,
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
		image,
		spec.Resources,
		[]string{"bash", "-c", "bundle exec puma -C config/puma.rb"},
		spec.EnvFrom,
		[]corev1.ContainerPort{
			{Name: "http", ContainerPort: 3000, Protocol: "TCP"},
		},
		&corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
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
	appName string,
	component componentType,
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
		if deployAnnotations == nil {
			deployAnnotations = map[string]string{}
		}
		for k, v := range deployAnnotations {
			deployAnnotations[k] = v
		}

		if deployLabels == nil {
			deployLabels = map[string]string{}
		}
		for k, v := range deployLabels {
			deployLabels[k] = v
		}

		selector := map[string]string{
			"app.kubernetes.io/name":      appName,
			"app.kubernetes.io/component": string(component),
			"app.kubernetes.io/part-of":   "mastodon",
		}
		deploy.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: selector,
		}

		podLabels := map[string]string{
			labelMastodonServer: server.GetName(),
			labelDeployImage:    encodeDeploymentImage(image),
		}
		for k, v := range selector {
			podLabels[k] = v
		}
		deploy.Spec.Template.SetLabels(podLabels)

		deploy.Spec.Replicas = &replicas
		deploy.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:           string(component),
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
	ctx context.Context, name, namespace string, kind jobType,
) (jobStatusType, error) {
	var job batchv1.Job
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      buildJobName(kind, name),
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
) (deployStatusType, *ImageMap, error) {
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

func (r *MastodonServerReconciler) deleteJob(ctx context.Context, name, namespace string) error {
	var job batchv1.Job
	if err := r.Client.Get(
		ctx,
		types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		&job,
	); err != nil {
		return err
	}

	propagationPolicy := metav1.DeletePropagationBackground
	return r.Client.Delete(ctx, &job, &client.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
}

//nolint:gocyclo
func decideWhatToDo(k8sStatus *k8sStatus) (whatToDoType, error) {
	mig := k8sStatus.migratingImageMap
	cur := k8sStatus.currentImageMap
	spec := k8sStatus.specImageMap

	preJNotFound := k8sStatus.preMigrationJobStatus == jobStatusNotFound
	preJCompleted := k8sStatus.preMigrationJobStatus == jobStatusCompleted
	preJFailed := k8sStatus.preMigrationJobStatus == jobStatusFailed
	preJNotCompleted := k8sStatus.preMigrationJobStatus == jobStatusNotCompleted
	postJNotFound := k8sStatus.postMigrationJobStatus == jobStatusNotFound
	postJCompleted := k8sStatus.postMigrationJobStatus == jobStatusCompleted
	postJFailed := k8sStatus.postMigrationJobStatus == jobStatusFailed
	postJNotCompleted := k8sStatus.postMigrationJobStatus == jobStatusNotCompleted
	dNotFound := k8sStatus.deploymentsStatus == deployStatusNotFound
	dReady := k8sStatus.deploymentsStatus == deployStatusReady
	dNotReady := k8sStatus.deploymentsStatus == deployStatusNotReady

	switch {
	case preJNotFound && postJNotFound && dNotFound && mig != nil: // S1
		fallthrough
	case preJCompleted && postJNotFound && dReady && mig != nil && cur.Equals(mig): // S31
		return shouldCreatePostMigrationJob, nil

	case preJNotFound && postJNotFound && dNotFound && mig == nil: // S2
		fallthrough
	case preJNotFound && postJNotFound && (dReady || dNotReady) && mig == nil && !cur.Equals(spec): // S33
		return shouldSetMigratingStatus, nil

	case preJNotFound && postJNotFound && (dReady || dNotReady) && mig != nil && cur.Equals(mig): // S5
		return shouldUnsetMigratingStatus, nil

	case preJNotFound && postJNotFound && (dReady || dNotReady) && mig != nil && !cur.Equals(mig): // S6
		return shouldCreatePreMigrationJob, nil

	case preJNotFound && postJNotFound && (dReady || dNotReady) && mig == nil && cur.Equals(spec): // S7
		return shouldCreateOrUpdateDeploysWithSpec, nil

	case preJNotFound && postJCompleted && dNotFound && mig != nil: // S8
		fallthrough
	case preJCompleted && postJNotFound && dReady && mig != nil && !cur.Equals(mig): // S20
		fallthrough
	case preJCompleted && postJNotFound && dNotReady && mig != nil && !cur.Equals(mig): // S20
		fallthrough
	case preJCompleted && postJNotFound && dNotReady && mig != nil && cur.Equals(mig): // S32
		return shouldCreateOrUpdateDeploysWithMigratingImages, nil

	case preJNotFound && postJCompleted && (dReady || dNotReady) && mig != nil && cur.Equals(mig): // S12
		fallthrough
	case postJFailed: // S30
		return shouldDeletePostMigrationJob, nil

	case preJCompleted && postJCompleted && (dReady || dNotReady) && mig != nil && cur.Equals(mig): // S26
		fallthrough
	case preJFailed: // S29
		return shouldDeletePreMigrationJob, nil

	case preJNotCompleted || postJNotCompleted: // S34
		return shouldDoNothing, nil
	}

	return -1, errors.New("unknown current status")
}
