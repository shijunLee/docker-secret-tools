package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"net"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/golang/glog"
	v1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/shijunLee/docker-secret-tools/pkg/config"
	"github.com/shijunLee/docker-secret-tools/pkg/utils"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

const (
	mutatingWebhookConfigurationName = "docker-secret-tools-mutating-webhook"
	mutatingWebhookName              = "docker-secret-tools"
	configName                       = "docker-secret-tools.shijunlee.net"
)

type JSONPath struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

//Server kubernetes Webhook server
type Server struct {
	server            *http.Server
	client            client.Client
	log               logr.Logger
	dockerSecretNames []string
	serviceName       string
	port              int
	restConfig        *rest.Config
	TLSPrivateKey     []byte
	TLSCert           []byte
	autoTLS           bool
	rootCA            string
	privateKeyFile    string
	certFile          string
}

//NewServer create a new webhook http server
func NewServer(mgr ctrl.Manager, serverConfig *config.Config) *Server {
	fmt.Println("create new server")
	serverInstance := &Server{
		client:            mgr.GetClient(),
		log:               mgr.GetLogger(),
		dockerSecretNames: serverConfig.DockerSecretNames,
		port:              serverConfig.ServerPort,
		serviceName:       serverConfig.ServiceName,
		restConfig:        mgr.GetConfig(),
		autoTLS:           serverConfig.AutoTLS,
		rootCA:            serverConfig.RootCA,
		privateKeyFile:    serverConfig.PrivateKeyFile,
		certFile:          serverConfig.CertFile,
	}
	fmt.Println("auto tls", serverConfig.AutoTLS)
	if serverConfig.AutoTLS {
		//get tls fail app can not start
		fmt.Println("start auto tls")
		privateKey, cert, err := serverInstance.createTLSConfig(context.TODO())
		if err != nil {
			serverInstance.log.Error(err, "get server instance cert error")
			panic(err)
		}
		fmt.Println(string(cert))
		serverInstance.TLSCert = cert
		serverInstance.TLSPrivateKey = privateKey
	}

	var httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", serverConfig.ServerPort),
		Handler: serverInstance,
	}
	serverInstance.server = httpServer
	return serverInstance
}

//Start start the webhook server
func (s *Server) Start(ctx context.Context) {

	// webhook 创建失败，应用立即失败，否则无法使用
	err := s.createAdmissionWebhook(ctx)
	if err != nil {
		s.log.Error(err, "create admission webhook error")
		panic(err)
	}
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

	var ln net.Listener
	var cert tls.Certificate
	if s.autoTLS {
		cert, err = tls.X509KeyPair(s.TLSCert, s.TLSPrivateKey)
		if err != nil {
			panic(err)
		}

	} else {
		cert, err = tls.LoadX509KeyPair(s.certFile, s.privateKeyFile)
		if err != nil {
			panic(err)
		}
	}
	var tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	ln, err = tls.Listen("tcp", s.server.Addr, tlsConfig)
	if err != nil {
		panic(err)
	}
	s.log.Error(s.server.Serve(ln), "web hook server error")
}

func (s *Server) createTLSConfig(ctx context.Context) (privateKey []byte, cert []byte, err error) {
	fmt.Println("start createTLSConfig")
	var secretNotFound = false
	var secret = &corev1.Secret{}
	var currentNamespace = utils.GetCurrentNameSpace()
	err = s.client.Get(ctx, types.NamespacedName{Namespace: currentNamespace, Name: mutatingWebhookConfigurationName}, secret)
	if err != nil && !k8serrors.IsNotFound(err) {
		s.log.Error(err, "get tls secret error")
		return nil, nil, err
	} else if k8serrors.IsNotFound(err) {
		secretNotFound = true
	}

	if !secretNotFound {
		privateKey = secret.Data["tls.key"]
		cert = secret.Data["tls.crt"]
		return
	}

	privateKey, cert, err = utils.CreateApproveTLSCert(ctx, s.restConfig, &utils.CertConfig{
		CertName:     s.serviceName,
		CertType:     utils.ServingCert,
		CommonName:   fmt.Sprintf("%s.%s.svc", s.serviceName, currentNamespace),
		Organization: []string{s.serviceName},
		DNSName: []string{
			"127.0.0.1",
			s.serviceName,
			fmt.Sprintf("%s.%s", s.serviceName, currentNamespace),
			fmt.Sprintf("%s.%s.svc", s.serviceName, currentNamespace),
			fmt.Sprintf("%s.%s.svc.cluster", s.serviceName, currentNamespace),
			fmt.Sprintf("%s.%s.svc.cluster.local", s.serviceName, currentNamespace),
		},
	})
	if err != nil {
		s.log.Error(err, "get private key and  cert error")
		return nil, nil, err
	}
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mutatingWebhookConfigurationName,
			Namespace: currentNamespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": privateKey,
			"tls.crt": cert,
		},
	}
	err = s.client.Create(ctx, secret)
	if err != nil {
		s.log.Error(err, "create tls secret error")
		return nil, nil, err
	}
	return
}

func (s *Server) createAdmissionWebhook(ctx context.Context) error {
	var scope = admissionregistrationv1.AllScopes
	var mutatingPath = "/mutate"
	var caBundle []byte
	var err error
	if s.autoTLS {
		caBundle, err = utils.GetKubernetesCA(ctx, s.client)
		if err != nil {
			s.log.Error(err, "get kubernetesCA error")
			return err
		}
	}

	mutatingWebhookConfiguration := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err = s.client.Get(ctx, types.NamespacedName{Namespace: utils.GetCurrentNameSpace(), Name: mutatingWebhookName}, mutatingWebhookConfiguration)
	if err != nil && !k8serrors.IsNotFound(err) {
		s.log.Error(err, "get mutatingWebhook error")
		return err
	} else if k8serrors.IsNotFound(err) {
		mutatingWebhookConfiguration.ObjectMeta = metav1.ObjectMeta{
			Name:      mutatingWebhookName,
			Namespace: utils.GetCurrentNameSpace(),
		}
		var failurePolicy = admissionregistrationv1.Ignore
		var sideEffectsConfig = admissionregistrationv1.SideEffectClassNone
		mutatingWebhookConfiguration.Webhooks = []admissionregistrationv1.MutatingWebhook{
			{
				Name:                    configName,
				SideEffects:             &sideEffectsConfig,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				FailurePolicy:           &failurePolicy,
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"", "apps"},
							APIVersions: []string{"*"},
							Resources: []string{
								"deployments",
								"daemonsets",
								"replicasets",
								"pods",
							},
							Scope: &scope,
						},
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: utils.GetCurrentNameSpace(),
						Name:      s.serviceName,
						Path:      &mutatingPath,
					},
				},
			},
		}
		if len(caBundle) > 0 {
			mutatingWebhookConfiguration.Webhooks[0].ClientConfig.CABundle = caBundle
		}
		err = s.client.Create(ctx, mutatingWebhookConfiguration)
		if err != nil {
			s.log.Error(err, "create mutatingWebhook error")
			return err
		}
	}
	oldCaBundle := mutatingWebhookConfiguration.Webhooks[0].ClientConfig.CABundle
	if !bytes.Equal(oldCaBundle, caBundle) {
		mutatingWebhookConfiguration.Webhooks[0].ClientConfig.CABundle = caBundle
		err = s.client.Update(ctx, mutatingWebhookConfiguration)
		if err != nil {
			s.log.Error(err, "update mutatingWebhook error")
			return err
		}
	}
	return nil
}

//ServeHTTP the http serve process
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/live" {
		w.WriteHeader(200)
		return
	}
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		s.log.Info("empty body")
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

	var admissionResponse *v1.AdmissionResponse
	ar := v1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		admissionResponse = &v1.AdmissionResponse{
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

	admissionReview := v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
	}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}
	resp, err := json.Marshal(admissionReview)
	s.log.Info("resp info", "Resp", string(resp))
	if err != nil {
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {

		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// validate deployments and services
func (s *Server) validate(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Allowed: true,
	}
}

// main mutation process
func (s *Server) mutate(ctx context.Context, ar *v1.AdmissionReview) *v1.AdmissionResponse {

	req := ar.Request
	s.log.Info("get mutate event", "AdmissionReview", ar, req.Kind.Kind, req.Kind.Group, req.Name, req.Namespace)
	var patchBytes []byte
	if req.Operation == v1.Connect || req.Operation == v1.Delete {
		return &v1.AdmissionResponse{
			Allowed: true,
		}
	}
	switch req.Kind.Kind {
	case "Deployment", "DaemonSet", "ReplicaSet", "Pod":
		jsonOrYamlData := req.Object.Raw
		jsonString := ""
		var rawString = string(jsonOrYamlData)
		s.log.Info("json data info", "JSON data", rawString)
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
		s.log.Info("json json String", "JSON  String", jsonString)
		imageList, err := utils.GetImageFromJSON(ctx, jsonString)
		if err != nil {
			s.log.Error(err, "get image from data error")
			break
		}
		s.log.Info("imageList", "imageList", imageList)
		if len(imageList) == 0 {
			s.log.Info("imageList not found")
			break
		}
		imageSecrets := s.getImagesSecrets(ctx, imageList)
		s.log.Info("get image secrets", "imageSecrets", imageSecrets)
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
			} else {
				replaceImageSecrets = append(replaceImageSecrets, item.Name)
			}
		}
		s.log.Info("get replace Image Secrets", "replaceImageSecrets", replaceImageSecrets)
		if len(replaceImageSecrets) > 0 {
			patchBytes = applySecret([]byte(jsonString), req.Kind.Kind, replaceImageSecrets)
			s.log.Info("patch data", "patch", string(patchBytes))
		}
	default:
		s.log.Info("return admission for kind not support")
		return &v1.AdmissionResponse{
			Allowed: true,
			UID:     req.UID,
		}
	}

	if len(patchBytes) > 0 {
		s.log.Info("return admission patch data", "patch", string(patchBytes))
		return &v1.AdmissionResponse{
			UID:     req.UID,
			Allowed: true,
			Patch:   patchBytes,
			PatchType: func() *v1.PatchType {
				pt := v1.PatchTypeJSONPatch
				return &pt
			}(),
		}
	} else {
		s.log.Info("return no patch for support")
		return &v1.AdmissionResponse{
			Allowed: true,
			UID:     req.UID,
		}
	}
}

func getImages(data []byte, kind string) []string {
	var result []string
	var podInfo = getPodTemplate(data, kind)
	if podInfo == nil {
		return nil
	}
	for _, c := range podInfo.Containers {
		result = append(result, c.Image)
	}
	return result
}

func applySecret(data []byte, kind string, secrets []string) []byte {
	switch kind {
	case "Deployment":
		var deployment = &appsv1.Deployment{}
		err := json.Unmarshal(data, deployment)
		if err != nil {
			return nil
		}
		for _, item := range secrets {
			deployment.Spec.Template.Spec.ImagePullSecrets = append(deployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: item})
		}
		newDeployment := deployment.DeepCopy()
		for _, item := range secrets {
			newDeployment.Spec.Template.Spec.ImagePullSecrets = append(newDeployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: item})
		}
		newDeploymentData, err := json.Marshal(newDeployment)
		if err != nil {
			return nil
		}
		resultData, err := strategicpatch.CreateTwoWayMergePatch(data, newDeploymentData, appsv1.Deployment{})
		if err != nil {
			return nil
		}
		return resultData
	case "DaemonSet":
		var ds = &appsv1.DaemonSet{}
		err := json.Unmarshal(data, ds)
		if err != nil {
			return nil
		}
		newDS := ds.DeepCopy()
		for _, item := range secrets {
			newDS.Spec.Template.Spec.ImagePullSecrets = append(newDS.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: item})
		}
		newDSData, err := json.Marshal(newDS)
		if err != nil {
			return nil
		}
		resultData, err := strategicpatch.CreateTwoWayMergePatch(data, newDSData, appsv1.DaemonSet{})
		if err != nil {
			return nil
		}
		return resultData
	case "StatefulSet":
		var sts = &appsv1.StatefulSet{}
		err := json.Unmarshal(data, sts)
		if err != nil {
			return nil
		}
		newSTS := sts.DeepCopy()
		for _, item := range secrets {
			newSTS.Spec.Template.Spec.ImagePullSecrets = append(newSTS.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: item})
		}
		newSTSData, err := json.Marshal(newSTS)
		if err != nil {
			return nil
		}
		resultData, err := strategicpatch.CreateTwoWayMergePatch(data, newSTSData, appsv1.StatefulSet{})
		if err != nil {
			return nil
		}
		return resultData
	case "ReplicaSet":
		var rs = &appsv1.ReplicaSet{}
		err := json.Unmarshal(data, rs)
		if err != nil {
			return nil
		}
		newRS := rs.DeepCopy()
		for _, item := range secrets {
			newRS.Spec.Template.Spec.ImagePullSecrets = append(newRS.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: item})
		}
		newRSData, err := json.Marshal(newRS)
		if err != nil {
			return nil
		}
		resultData, err := strategicpatch.CreateTwoWayMergePatch(data, newRSData, appsv1.ReplicaSet{})
		if err != nil {
			return nil
		}
		return resultData
	case "Pod":
		var pod = &corev1.Pod{}
		err := json.Unmarshal(data, pod)
		if err != nil {
			return data
		}
		newPod := pod.DeepCopy()
		for _, item := range secrets {
			newPod.Spec.ImagePullSecrets = append(newPod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: item})
		}
		newPodData, err := json.Marshal(newPod)
		if err != nil {
			return nil
		}
		resultData, err := strategicpatch.CreateTwoWayMergePatch(data, newPodData, corev1.Pod{})
		if err != nil {
			return nil
		}
		return resultData
	}
	return data
}

func getPodTemplate(data []byte, kind string) *corev1.PodSpec {
	switch kind {
	case "Deployment":
		var deployment = &appsv1.Deployment{}
		err := json.Unmarshal(data, deployment)
		if err != nil {
			return nil
		}
		return &(deployment.Spec.Template.Spec)
	case "DaemonSet":
		var ds = &appsv1.DaemonSet{}
		err := json.Unmarshal(data, ds)
		if err != nil {
			return nil
		}
		return &(ds.Spec.Template.Spec)
	case "StatefulSet":
		var sts = &appsv1.StatefulSet{}
		err := json.Unmarshal(data, sts)
		if err != nil {
			return nil
		}
		return &(sts.Spec.Template.Spec)
	case "ReplicaSet":
		var rs = &appsv1.ReplicaSet{}
		err := json.Unmarshal(data, rs)
		if err != nil {
			return nil
		}
		return &(rs.Spec.Template.Spec)
	case "Pod":
		var pod = &corev1.Pod{}
		err := json.Unmarshal(data, pod)
		if err != nil {
			return nil
		}
		return &(pod.Spec)
	}

	return nil
}

func (s *Server) getImagesSecrets(ctx context.Context, images []string) []corev1.Secret {
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

func (s *Server) getSecretAuthRegistry(ctx context.Context) map[string][]corev1.Secret {
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
