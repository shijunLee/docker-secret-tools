package webhook

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

type Server struct {
	server *http.Server
}

func (whsvr *Server) serve(w http.ResponseWriter, r *http.Request) {
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
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		fmt.Println(r.URL.Path)
		if r.URL.Path == "/mutate" {
			admissionResponse = whsvr.mutate(&ar)
		} else if r.URL.Path == "/validate" {
			admissionResponse = whsvr.validate(&ar)
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
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// validate deployments and services
func (whsvr *Server) validate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		//availableLabels                 map[string]string
		//objectMeta                      *metav1.ObjectMeta
		//resourceNamespace, resourceName string
		resourceName string
	)

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	//switch req.Kind.Kind {
	//case "Deployment":
	//	var deployment appsv1.Deployment
	//	if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
	//		glog.Errorf("Could not unmarshal raw object: %v", err)
	//		return &v1beta1.AdmissionResponse{
	//			Result: &metav1.Status{
	//				Message: err.Error(),
	//			},
	//		}
	//	}
	//	resourceName, resourceNamespace, objectMeta = deployment.Name, deployment.Namespace, &deployment.ObjectMeta
	//	availableLabels = deployment.Labels
	//case "Service":
	//	var service corev1.Service
	//	if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
	//		glog.Errorf("Could not unmarshal raw object: %v", err)
	//		return &v1beta1.AdmissionResponse{
	//			Result: &metav1.Status{
	//				Message: err.Error(),
	//			},
	//		}
	//	}
	//	resourceName, resourceNamespace, objectMeta = service.Name, service.Namespace, &service.ObjectMeta
	//	availableLabels = service.Labels
	//}
	//
	//if !validationRequired(ignoredNamespaces, objectMeta) {
	//	glog.Infof("Skipping validation for %s/%s due to policy check", resourceNamespace, resourceName)
	//	return &v1beta1.AdmissionResponse{
	//		Allowed: true,
	//	}
	//}
	//
	//allowed := true
	//var result *metav1.Status
	//glog.Info("available labels:", availableLabels)
	//glog.Info("required labels", requiredLabels)
	//for _, rl := range requiredLabels {
	//	if _, ok := availableLabels[rl]; !ok {
	//		allowed = false
	//		result = &metav1.Status{
	//			Reason: "required labels are not set",
	//		}
	//		break
	//	}
	//}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
		//	Result:  result,
	}
}

// main mutation process
func (whsvr *Server) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   []byte{}, // patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
