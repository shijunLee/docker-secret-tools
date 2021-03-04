package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/golang/glog"
	"github.com/shijunLee/docker-secret-tools/pkg/utils"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

type server struct {
	server            *http.Server
	client            client.Client
	log               logr.Logger
	dockerSecretNames []string
}

//NewServer create a new webhook http server
func NewServer(mgr ctrl.Manager, dockerSecretNames []string) *server {
	serverInstance := &server{
		client:            mgr.GetClient(),
		log:               mgr.GetLogger(),
		dockerSecretNames: dockerSecretNames,
	}
	var httpServer = &http.Server{
		Addr:    ":8080",
		Handler: serverInstance,
	}
	serverInstance.server = httpServer
	return serverInstance
}

//Start start the webhook server
func (s *server) Start() {
	defer func() {
		// 发生宕机时，获取panic传递的上下文并打印
		err := recover()
		errInfo, ok := err.(error)
		if ok {
			s.log.Error(errInfo, "webhook recover from panic error")
		} else {
			fmt.Println("error:", err)
		}
	}()
	// TODO: create tls key and webhook registry for request
	s.log.Error(s.server.ListenAndServe(), "web hook server error")
}

//ServeHTTP the http serve process
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		fmt.Println(r.URL.Path)
		if r.URL.Path == "/mutate" {
			admissionResponse = s.mutate(r.Context(), &ar)
		} else if r.URL.Path == "/validate" {
			admissionResponse = s.validate(&ar)
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}

	if _, err := w.Write(resp); err != nil {

		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// validate deployments and services
func (s *server) validate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		//	Result:  result,
	}
}

// main mutation process
func (s *server) mutate(ctx context.Context, ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var patchBytes []byte
	if req.Operation == v1beta1.Connect || req.Operation == v1beta1.Delete {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	switch req.Kind.Kind {
	case "Deployment", "DaemonSet", "ReplicaSet", "Pod":
		jsonOrYamlData := req.Object.Raw
		jsonString := ""
		var rawString = string(jsonOrYamlData)
		if strings.HasPrefix(strings.TrimLeft(rawString, " "), "{") {
			jsonString = rawString
		} else {
			jsonData, err := yaml.YAMLToJSON(jsonOrYamlData)
			if err != nil {
				s.log.Error(err, "get raw data from not json error", "RawData", rawString)
				break
			}
			jsonString = string(jsonData)
		}
		imageList, err := utils.GetImageFromJson(ctx, jsonString)
		if err != nil {
			s.log.Error(err, "get image from data error")
			break
		}
		if len(imageList) == 0 {
			break
		}
		imageSecrets := s.getImagesSecrets(ctx, imageList)
		var replaceImageSecrets []string
		for _, item := range imageSecrets {
			var secret = &corev1.Secret{}
			err = s.client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: item.Name}, secret)
			if err != nil && k8serrors.IsNotFound(err) {
				item.Namespace = req.Namespace
				err = s.client.Create(ctx, &item)
				if err != nil {
					s.log.Error(err, "create secret error", "SecretName", item.Name)
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
			switch req.Kind.Kind {
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
			patchBytes, err = json.Marshal(secretMaps)
			if err != nil {
				s.log.Error(err, "convert secret to json error")
			}
		}
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	if len(patchBytes) > 0 {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
			Patch:   patchBytes,
			PatchType: func() *v1beta1.PatchType {
				pt := v1beta1.PatchTypeJSONPatch
				return &pt
			}(),
		}
	} else {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

}

func (s *server) getImagesSecrets(ctx context.Context, images []string) []corev1.Secret {
	var registrySecrets = s.getSecretAuthRegistry(ctx)
	var result = []corev1.Secret{}

	for k, v := range registrySecrets {
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

func (s *server) getSecretAuthRegistry(ctx context.Context) map[string][]corev1.Secret {
	var result = map[string][]corev1.Secret{}
	var secrets = utils.GetDockerSecrets(ctx, s.client, s.log, s.dockerSecretNames)
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
				s.log.Error(err, "unmarshal docker secret to docker config error")
			}
		}
	}
	return result
}
