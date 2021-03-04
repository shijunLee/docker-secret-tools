package controller

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	"github.com/shijunLee/docker-secret-tools/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
)

type WorkloadReconciler struct {
	client.Client
	Log               logr.Logger
	Object            client.Object
	NotManagerOwners  []string
	DockerSecretNames []string
}

func (w *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var object = &unstructured.Unstructured{}
	object.SetGroupVersionKind(w.Object.GetObjectKind().GroupVersionKind())
	err := w.Client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, object)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, err
		}
	}
	jsonData, err := object.MarshalJSON()
	if err != nil {
		w.Log.Error(err, "get json data error")
	}
	imageList, err := utils.GetImageFromJson(ctx, string(jsonData))
	if err != nil {
		w.Log.Error(err, "get image from data error")
		return ctrl.Result{}, nil
	}
	if len(imageList) == 0 {
		return ctrl.Result{}, nil
	}
	imageSecrets := w.getImagesSecrets(ctx, imageList)
	var replaceImageSecrets []string
	for _, item := range imageSecrets {
		var secret = &corev1.Secret{}
		err = w.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: item.Name}, secret)
		if err != nil && k8serrors.IsNotFound(err) {
			item.Namespace = req.Namespace
			err = w.Client.Create(ctx, &item)
			if err != nil {
				w.Log.Error(err, "create secret error", "SecretName", item.Name)
			} else {
				replaceImageSecrets = append(replaceImageSecrets, item.Name)
			}
		}
	}
	if len(replaceImageSecrets) > 0 {
		var secretListKV []map[string]string
		for _, secret := range replaceImageSecrets {
			secretListKV = append(secretListKV, map[string]string{"name": secret})
		}
		var secretMaps map[string]interface{}
		switch object.GetKind() {
		case "Pod":
			secretMaps = map[string]interface{}{
				"spec": map[string]interface{}{
					"imagePullSecrets": secretListKV,
				},
			}

		default:
			secretMaps = map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"imagePullSecrets": secretListKV,
						},
					},
				},
			}
		}
		mergePatch, err := json.Marshal(secretMaps)
		if err != nil {
			w.Log.Error(err, "convert secret to json error")
		}
		err = w.Patch(ctx, object, client.RawPatch(types.StrategicMergePatchType, mergePatch))
		if err != nil {
			w.Log.Error(err, "patch object secret error", "Group", object.GroupVersionKind().Group,
				"Version", object.GroupVersionKind().Version, "Kind", object.GroupVersionKind().Kind, "Name", object.GetName(),
				"Namespace", object.GetNamespace())
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (w *WorkloadReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(w.Object).WithEventFilter(predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return w.filterEventObject(ctx, event.Object)
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
	}).Complete(w)
}

func (w *WorkloadReconciler) getImagesSecrets(ctx context.Context, images []string) []corev1.Secret {
	var regsitrySecrets = w.getSecretAuthRegistry(ctx)
	var result = []corev1.Secret{}

	for k, v := range regsitrySecrets {
		for _, image := range images {
			imagePathURLSplits := strings.Split(image, ":")
			if len(imagePathURLSplits) == 0 {
				continue
			}
			imagePathSplits := strings.Split(imagePathURLSplits[0], "/")
			if len(imagePathSplits) == 0 {
				continue
			}
			imageHost := imagePathSplits[0]
			if imageHost == k {
				for _, item := range v {
					result = append(result, item)
				}
			}
		}
	}
	return result
}

func (w *WorkloadReconciler) getSecretAuthRegistry(ctx context.Context) map[string][]corev1.Secret {
	var result = map[string][]corev1.Secret{}
	var secrets = utils.GetDockerSecrets(ctx, w.Client, w.Log, w.DockerSecretNames)
	for _, item := range secrets {
		configData, ok := item.Data[".dockerconfigjson"]
		if ok {
			var dockerSecrets = &utils.DockerSecrets{}
			err := json.Unmarshal(configData, dockerSecrets)
			if err == nil {
				for key, _ := range dockerSecrets.Auths {
					if values, ok := result[key]; ok {
						values = append(values, *item)
						result[key] = values
					} else {
						var values []corev1.Secret
						values = append(values, *item)
						result[key] = values
					}
				}
			} else {
				w.Log.Error(err, "unmarshal docker secret to docker config error")
			}
		}
	}
	return result
}

func (w *WorkloadReconciler) filterEventObject(ctx context.Context, object client.Object) bool {
	ownerReference := object.GetOwnerReferences()
	if ownerReference != nil && len(ownerReference) > 0 {
		for _, item := range ownerReference {
			if item.APIVersion == "apps/v1" {
				return false
			}
			for _, notManagerOwner := range w.NotManagerOwners {
				if item.APIVersion == notManagerOwner {
					return false
				}
			}
		}
	}
	return true
}
