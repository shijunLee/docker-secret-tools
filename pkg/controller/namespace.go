package controller

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/shijunLee/docker-secret-tools/pkg/utils"
)

type NamespaceReconciler struct {
	client.Client
	Log               logr.Logger
	DockerSecretNames []string
}

//Reconcile auto create secret to new namespace
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var secrets = utils.GetDockerSecrets(ctx, r.Client, r.Log, r.DockerSecretNames)
	for _, item := range secrets {
		imagePullSecret := &corev1.Secret{}
		var namespace = req.Name
		if namespace == "" {
			namespace = "default"
		}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: item.Name}, imagePullSecret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				var secret = *item

				secret.Namespace = namespace
				secret.ObjectMeta.ResourceVersion = ""
				err := r.Client.Create(ctx, &secret)
				if err != nil {
					r.Log.Error(err, "create secret to namespace error", "SecretName", secret.Name, "Namespace", req.Namespace)
					return ctrl.Result{}, err
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

func (w *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).WithEventFilter(predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
	}).Complete(w)
}
