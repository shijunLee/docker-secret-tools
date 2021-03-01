package controller

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-logr/logr"
	"github.com/shijunLee/docker-secret-tools/pkg/utils"
	"github.com/thedevsaddam/gojsonq"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/yaml"
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
	imageList, err := getImageFromJson(ctx, string(jsonData))
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
	var secrets = getDockerSecrets(ctx, w.Client, w.Log, w.DockerSecretNames)
	for _, item := range secrets {
		configData, ok := item.Data[".dockerconfigjson"]
		if ok {
			var dockerSecrets = &DockerSecrets{}
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
func getDockerSecrets(ctx context.Context, mgrClient client.Client, logger logr.Logger, dockerSecretNames []string) (imageSecrets []*corev1.Secret) {
	for _, item := range dockerSecretNames {
		var secret = &corev1.Secret{}
		err := mgrClient.Get(ctx, types.NamespacedName{Namespace: utils.GetCurrentNameSpace(), Name: item}, secret)
		if err != nil {
			continue
		}
		if secret.Type == "kubernetes.io/dockercfg" {
			logger.Info("Not support dockercfg docker secret", "SecretName", item)
			continue
		}
		if secret.Type != "kubernetes.io/dockerconfigjson" {
			continue
		}
		imageSecrets = append(imageSecrets, secret)
	}
	return
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
func getImageFromJson(ctx context.Context, jsonString string) (imageList []string, err error) {
	data := gojsonq.New().FromString(jsonString).Find("spec.template.spec.containers")
	if data == nil {
		data = gojsonq.New().FromString(jsonString).Find("spec.containers")
	}
	dataInterface, ok := data.([]interface{})
	if !ok {
		return nil, errors.New("convert type to []interface error")
	}
	for _, item := range dataInterface {
		if mapDatas, ok := item.(map[string]interface{}); ok {
			if image, ok := mapDatas["image"]; ok {
				imageList = append(imageList, image.(string))
			}
		} else {
			return nil, errors.New("convert type to map[string]interface{} error")
		}
	}
	return
}

func getImageFromYaml(ctx context.Context, yamlString string) (imageList []string, err error) {
	jsondata, err := yaml.YAMLToJSON([]byte(yamlString))
	if err != nil {
		return nil, err
	}
	return getImageFromJson(ctx, string(jsondata))
}

type DockerSecrets struct {
	Auths map[string]DockerAuth `json:"auths,omitempty"`
}

type DockerAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`
}
