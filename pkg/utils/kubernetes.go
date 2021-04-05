package utils

import (
	"context"
	"errors"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"

	"github.com/go-logr/logr"
	"github.com/thedevsaddam/gojsonq"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

//CreateApproveTLSCert create TLS cert with kubernetes Certificate Signing
func CreateApproveTLSCert(ctx context.Context, restConfig *rest.Config, config *CertConfig) (privateKeyData []byte, certificateData []byte, err error) {
	certClient := kubernetes.NewForConfigOrDie(restConfig).CertificatesV1().CertificateSigningRequests()

	request, err := certClient.Get(ctx, "docker-secret-tools", metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, nil, err
	} else if err == nil {
		err = certClient.Delete(ctx, "docker-secret-tools", metav1.DeleteOptions{})
		if err != nil {
			return nil, nil, err
		}
	}
	privateKey, certificateRequest, err := CreateCertificateRequest(config)
	if err != nil {
		return nil, nil, err
	}
	var certificateRequestbytes = EncodeCertificateRequestPEM(certificateRequest)
	certificateSigningRequest := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docker-secret-tools",
			Namespace: GetCurrentNameSpace(),
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Groups: []string{
				"system:authenticated",
			},
			SignerName: "shijunlee.net/docker-tool",
			Request:    certificateRequestbytes,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageServerAuth,
			},
		},
	}
	_, err = certClient.Create(ctx, certificateSigningRequest, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}

	request, err = certClient.Get(ctx, "docker-secret-tools", metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	_, err = certClient.UpdateApproval(ctx, "docker-secret-tools", request, metav1.UpdateOptions{})
	if err != nil {
		return nil, nil, err
	}
	request, err = certClient.Get(ctx, "docker-secret-tools", metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	certificateData = request.Status.Certificate
	privateKeyData = EncodePrivateKeyPEM(privateKey)
	return privateKeyData, certificateData, nil
}
