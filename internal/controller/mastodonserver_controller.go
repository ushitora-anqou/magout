package controller

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	magoutv1 "github.com/ushitora-anqou/magout/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	domain              = "magout.anqou.net"
	labelMastodonServer = domain + "/mastodon-server"
	labelDeployImage    = domain + "/deploy-image"
)

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

	if err := r.createOrUpdateDeployments(ctx, &server); err != nil {
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

func (r *MastodonServerReconciler) createOrUpdateDeployments(
	ctx context.Context,
	server *magoutv1.MastodonServer,
) error {
	if err := r.createOrUpdateSidekiqDeployment(ctx, server); err != nil {
		return fmt.Errorf("failed to create or update sidekiq deployment: %w", err)
	}
	return nil
}

func (r *MastodonServerReconciler) createOrUpdateSidekiqDeployment(
	ctx context.Context,
	server *magoutv1.MastodonServer,
) error {
	logger := log.FromContext(ctx)
	spec := server.Spec.Sidekiq

	var deploy appsv1.Deployment
	deploy.SetName(fmt.Sprintf("%s-sidekiq", server.GetName()))
	deploy.SetNamespace(server.GetNamespace())

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		deploy.SetAnnotations(spec.Annotations)

		selector := map[string]string{
			"app.kubernetes.io/name":      "sidekiq",
			"app.kubernetes.io/component": "sidekiq",
			"app.kubernetes.io/part-of":   "mastodon",
		}
		deploy.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: selector,
		}

		labels := map[string]string{
			labelMastodonServer: server.GetName(),
			labelDeployImage:    "", // FIXME
		}
		for k, v := range selector {
			labels[k] = v
		}
		for k, v := range spec.Labels {
			labels[k] = v
		}
		deploy.SetLabels(labels)

		deploy.Spec.Replicas = &spec.Replicas
		deploy.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:      "sidekiq",
				Image:     spec.Image,
				Resources: spec.Resources,
				Command:   []string{"bash", "-c", "bundle exec sidekiq"},
			},
		}
		return ctrl.SetControllerReference(server, &deploy, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to create or update: %w", err)
	}
	if result != controllerutil.OperationResultNone {
		logger.Info(
			"create sidekiq deployment successfully",
			"name", deploy.GetName(),
			"namespace", deploy.GetNamespace(),
		)
	}

	return nil
}
