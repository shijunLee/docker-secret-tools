package utils

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-logr/logr"
	"github.com/thedevsaddam/gojsonq"
	certificatesv1 "k8s.io/api/certificates/v1"
	certificatesV1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/certificate/csr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const currentNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

//GetCurrentNameSpace get current pods run namespace
func GetCurrentNameSpace() string {
	currentNameSpace := os.Getenv("DEBUG_NAMESPACE")
	if currentNameSpace != "" {
		return currentNameSpace
	}
	_, err := os.Stat(currentNamespacePath)
	if err != nil {
		return currentNameSpace
	}
	data, err := ioutil.ReadFile(currentNamespacePath)
	if err != nil {
		return currentNameSpace
	}
	return string(data)
}

// GetImageFromJSON get image info from workload object json string
func GetImageFromJSON(ctx context.Context, jsonString string) (imageList []string, err error) {
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

//GetImageFromYaml get image from workload object yaml
func GetImageFromYaml(ctx context.Context, yamlString string) (imageList []string, err error) {
	jsondata, err := yaml.YAMLToJSON([]byte(yamlString))
	if err != nil {
		return nil, err
	}
	return GetImageFromJSON(ctx, string(jsondata))
}

//DockerSecrets docker secrets object
type DockerSecrets struct {
	Auths map[string]DockerAuth `json:"auths,omitempty"`
}

//DockerAuth docker registry auth info
type DockerAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

//GetDockerSecrets get docker secrets in dockerSecretNames
func GetDockerSecrets(ctx context.Context, mgrClient client.Client, logger logr.Logger, dockerSecretNames []string) (imageSecrets []*corev1.Secret) {
	for _, item := range dockerSecretNames {
		var secret = &corev1.Secret{}
		err := mgrClient.Get(ctx, types.NamespacedName{Namespace: GetCurrentNameSpace(), Name: item}, secret)
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

// GetKubernetesCA get current cluster ca
func GetKubernetesCA(ctx context.Context, c client.Client) ([]byte, error) {
	var result []byte
	secretList := &corev1.SecretList{}
	err := c.List(ctx, secretList, &client.ListOptions{Namespace: GetCurrentNameSpace()})
	if err != nil {
		return nil, err
	}
	for _, item := range secretList.Items {
		if item.Type == corev1.SecretTypeServiceAccountToken {
			if value, ok := item.Annotations["kubernetes.io/service-account.name"]; ok && value == "default" {
				result = item.Data["ca.crt"]
				return result, nil
			}
		}
	}
	return nil, errors.New("token not found")
}

var csrName = "docker-secret-tool-webhook.tool-test"

//CreateApproveTLSCert create TLS cert with kubernetes Certificate Signing
// Notice this is not work for the kubernetes not config cert sign config
func CreateApproveTLSCert(ctx context.Context, restConfig *rest.Config, config *CertConfig) (privateKeyData []byte, certificateData []byte, err error) {
	fmt.Println("start CreateApproveTLSCert")
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	var isSupportV1 = false
	_, err = kubeClient.ServerResourcesForGroupVersion("certificates.k8s.io/v1")
	if err == nil {
		isSupportV1 = true
	}
	if isSupportV1 {
		certClient := kubeClient.CertificatesV1().CertificateSigningRequests()
		_, err = certClient.Get(ctx, csrName, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, nil, err
		} else if err == nil {
			err = certClient.Delete(ctx, csrName, metav1.DeleteOptions{})
			if err != nil {
				return nil, nil, err
			}
		}
	} else {
		certClient := kubeClient.CertificatesV1beta1().CertificateSigningRequests()
		_, err = certClient.Get(ctx, csrName, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, nil, err
		} else if err == nil {
			err = certClient.Delete(ctx, csrName, metav1.DeleteOptions{})
			if err != nil {
				return nil, nil, err
			}
		}
	}

	privateKey, certificateRequest, err := CreateCertificateRequest(config)
	if err != nil {
		return nil, nil, err
	}
	var certificateRequestbytes = EncodeCertificateRequestPEM(certificateRequest)
	var usage = []certificatesv1.KeyUsage{
		certificatesv1.UsageServerAuth,
		certificatesv1.UsageClientAuth,
	}
	reqName, reqUID, err := csr.RequestCertificate(kubeClient, certificateRequestbytes, csrName, "kubernetes.io/legacy-unknown", usage, privateKey)
	if err != nil {
		return nil, nil, err
	}
	if isSupportV1 {
		certClient := kubeClient.CertificatesV1().CertificateSigningRequests()
		csrRequest, err := certClient.Get(ctx, csrName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}
		csrRequest.Status.Conditions = append(csrRequest.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
			Type:           certificatesv1.CertificateApproved,
			Reason:         "docker secret tools webhook",
			Message:        "This CSR was approved by docker-tools",
			LastUpdateTime: metav1.Now(),
			Status:         corev1.ConditionTrue,
		})
		_, err = certClient.UpdateApproval(ctx, csrName, csrRequest, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
	} else {
		certClient := kubeClient.CertificatesV1beta1().CertificateSigningRequests()
		csrRequest, err := certClient.Get(ctx, csrName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}
		csrRequest.Status.Conditions = append(csrRequest.Status.Conditions, certificatesV1beta1.CertificateSigningRequestCondition{
			Type:           certificatesV1beta1.CertificateApproved,
			Reason:         "test",
			Message:        "This CSR was approved by test",
			LastUpdateTime: metav1.Now(),
			Status:         corev1.ConditionTrue,
		})
		_, err = certClient.UpdateApproval(ctx, csrRequest, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
	}
	fmt.Println("start WaitForCertificate")
	data, err := csr.WaitForCertificate(ctx, kubeClient, reqName, reqUID)
	if err != nil {
		return nil, nil, err
	}
	certificateData = data
	privateKeyData = EncodePrivateKeyPEM(privateKey)
	return privateKeyData, certificateData, nil
}
