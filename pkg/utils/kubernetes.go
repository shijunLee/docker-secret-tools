package utils

import (
	"context"
	"errors"
	"github.com/go-logr/logr"
	"github.com/thedevsaddam/gojsonq"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const currentNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func GetCurrentNameSpace() string {
	currentNameSpace := os.Getenv("DEBUG_NAMESPACE")
	if currentNameSpace != "" {
		return currentNameSpace
	} else {
		_, err := os.Stat(currentNamespacePath)
		if err != nil {
			return currentNameSpace
		} else {
			data, err := ioutil.ReadFile(currentNamespacePath)
			if err != nil {
				return currentNameSpace
			} else {
				return string(data)
			}
		}
	}
}

func GetImageFromJson(ctx context.Context, jsonString string) (imageList []string, err error) {
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

func GetImageFromYaml(ctx context.Context, yamlString string) (imageList []string, err error) {
	jsondata, err := yaml.YAMLToJSON([]byte(yamlString))
	if err != nil {
		return nil, err
	}
	return GetImageFromJson(ctx, string(jsondata))
}

type DockerSecrets struct {
	Auths map[string]DockerAuth `json:"auths,omitempty"`
}

type DockerAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

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
