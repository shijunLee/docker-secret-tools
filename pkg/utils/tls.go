// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// CertType defines the type of the cert.
type CertType int

// CA error info
var (
	ErrCANotFound        = errors.New("no CA secret and ConfigMap found")
	ErrCAKeyAndCACertReq = errors.New("a CA key and CA cert need to be provided when requesting a custom CA")
	ErrInternal          = errors.New("internal error while generating TLS assets")
)

const (
	// ClientAndServingCert defines both client and serving cert.
	ClientAndServingCert CertType = iota
	// ServingCert defines a serving cert.
	ServingCert
	// ClientCert defines a client cert.
	ClientCert
)

// CertConfig configures how to generate the Cert.
type CertConfig struct {
	// CertName is the name of the cert.
	CertName string
	// Optional CertType. Serving, client or both; defaults to both.
	CertType CertType
	// Optional CommonName is the common name of the cert; defaults to "".
	CommonName string
	// Optional Organization is Organization of the cert; defaults to "".
	Organization []string

	DNSName []string
}

//GenerateCert returns a secret containing the TLS encryption key and cert,
// a ConfigMap containing the CA Certificate and a Secret containing the CA key or it
// returns a error incase something goes wrong. ca provate key ,root ca ,ca ,error
func GenerateCert(config *CertConfig) (*rsa.PrivateKey, *x509.Certificate, *x509.Certificate, error) {
	if err := verifyConfig(config); err != nil {
		return nil, nil, nil, err
	}
	// If no custom CAKey and CACert are provided we have to generate them
	caKey, err := newPrivateKey()
	if err != nil {
		return nil, nil, nil, err
	}
	caCert, err := newSelfSignedCACertificate(caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	key, err := newPrivateKey()
	if err != nil {
		return nil, nil, nil, err
	}
	cert, err := newSignedCertificate(config, key, caCert, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	return key, caCert, cert, nil

}

func verifyConfig(config *CertConfig) error {
	if config == nil {
		return errors.New("nil CertConfig not allowed")
	}
	if config.CertName == "" {
		return errors.New("empty CertConfig.CertName not allowed")
	}
	return nil
}

//ToAppSecretName create app secret name
func ToAppSecretName(kind, name, certName string) string {
	return strings.ToLower(kind) + "-" + name + "-" + certName
}

//ToCASecretAndConfigMapName create app ca secret name
func ToCASecretAndConfigMapName(kind, name string) string {
	return strings.ToLower(kind) + "-" + name + "-ca"
}

// getCASecretAndConfigMapInCluster gets CA secret and configmap of the given name and namespace.
// it only returns both if they are found and nil if both are not found. In the case if only one of them is found,
// then we error out because we expect either both CA secret and configmap exit or not.
//
// NOTE: both the CA secret and configmap have the same name with template `<cr-kind>-<cr-name>-ca` which is what the
// input parameter `name` refers to.
func getCASecretAndConfigMapInCluster(kubeClient kubernetes.Interface, name,
	namespace string) (*v1.Secret, *v1.ConfigMap, error) {
	hasConfigMap := true
	cm, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !apiErrors.IsNotFound(err) {
		return nil, nil, err
	}
	if apiErrors.IsNotFound(err) {
		hasConfigMap = false
	}

	hasSecret := true
	se, err := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !apiErrors.IsNotFound(err) {
		return nil, nil, err
	}
	if apiErrors.IsNotFound(err) {
		hasSecret = false
	}
	if hasConfigMap != hasSecret {
		// TODO: this case can happen if creating CA configmap succeeds and creating CA secret failed.
		//  We need to handle this case properly.
		return nil, nil, fmt.Errorf("expect either both ca configmap and secret both exist or not exist, "+
			" but got hasCAConfigmap==%v and hasCASecret==%v", hasConfigMap, hasSecret)
	}
	if !hasConfigMap {
		return nil, nil, nil
	}
	return se, cm, nil
}

func toKindNameNamespace(cr runtime.Object) (string, string, string, error) {
	a := meta.NewAccessor()
	k, err := a.Kind(cr)
	if err != nil {
		return "", "", "", err
	}
	n, err := a.Name(cr)
	if err != nil {
		return "", "", "", err
	}
	ns, err := a.Namespace(cr)
	if err != nil {
		return "", "", "", err
	}
	return k, n, ns, nil
}

// toTLSSecret returns a client/server "kubernetes.io/tls" secret.
func toTLSSecret(key *rsa.PrivateKey, cert *x509.Certificate, name string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			v1.TLSPrivateKeyKey: EncodePrivateKeyPEM(key),
			v1.TLSCertKey:       EncodeCertificatePEM(cert),
		},
		Type: v1.SecretTypeTLS,
	}
}

//CreateCertificateRequest create certificate request
func CreateCertificateRequest(config *CertConfig) (*rsa.PrivateKey, *x509.CertificateRequest, error) {
	if err := verifyConfig(config); err != nil {
		return nil, nil, err
	}
	caKey, err := newPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	certificateRequest, err := CreateCertificateTool(caKey, config)
	if err != nil {
		return nil, nil, err
	}

	return caKey, certificateRequest, nil
}
