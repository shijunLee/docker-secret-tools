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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math"
	"math/big"
	"time"
)

const (
	rsaKeySize   = 2048
	duration365d = time.Hour * 24 * 365 * 10
)

// newPrivateKey returns randomly generated RSA private key.
func newPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// EncodePrivateKeyPEM encodes the given private key pem and returns bytes (base64).
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// EncodeCertificatePEM encodes the given certificate pem and returns bytes (base64).
func EncodeCertificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// parsePEMEncodedCert parses a certificate from the given pemdata
func parsePEMEncodedCert(pemdata []byte) (*x509.Certificate, error) {
	decoded, _ := pem.Decode(pemdata)
	if decoded == nil {
		return nil, errors.New("no PEM data found")
	}
	return x509.ParseCertificate(decoded.Bytes)
}

// parsePEMEncodedPrivateKey parses a private key from given pemdata
func parsePEMEncodedPrivateKey(pemdata []byte) (*rsa.PrivateKey, error) {
	decoded, _ := pem.Decode(pemdata)
	if decoded == nil {
		return nil, errors.New("no PEM data found")
	}
	return x509.ParsePKCS1PrivateKey(decoded.Bytes)
}

// newSelfSignedCACertificate returns a self-signed CA certificate based on given configuration and private key.
// The certificate has one-year lease.
func newSelfSignedCACertificate(key *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := x509.Certificate{
		Subject: pkix.Name{
			Organization:       []string{"Kubernetes"},
			Locality:           []string{"SH"},
			OrganizationalUnit: []string{"CA"},
			Province:           []string{"SH"},
			Country:            []string{"CN"},
		},
		SerialNumber:          serial,
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// newSignedCertificate signs a certificate using the given private key, CA and returns a signed certificate.
// The certificate could be used for both client and server auth.
// The certificate has one-year lease.
func newSignedCertificate(cfg *CertConfig, key *rsa.PrivateKey, caCert *x509.Certificate,
	caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	eku := []x509.ExtKeyUsage{}
	switch cfg.CertType {
	case ClientCert:
		eku = append(eku, x509.ExtKeyUsageClientAuth)
	case ServingCert:
		eku = append(eku, x509.ExtKeyUsageServerAuth)
	case ClientAndServingCert:
		eku = append(eku, x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth)
	}
	organization := cfg.Organization
	if organization == nil || len(organization) == 0 {
		organization = []string{"jdcloud"}
	}
	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:         cfg.CommonName,
			Organization:       organization,
			Locality:           []string{"BJ"},
			OrganizationalUnit: []string{},
			Province:           []string{"BJ"},
			Country:            []string{"CN"},
		},
		DNSNames:     cfg.DNSName,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  eku,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// EncodeCertificateRequestPEM encodes the given certificate pem and returns bytes (base64).
func EncodeCertificateRequestPEM(cert *x509.CertificateRequest) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: cert.Raw,
	})
}

//CreatePrivateKey create a PrivateKey
func CreatePrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

//CreateCertificateTool create a CertificateRequest
func CreateCertificateTool(privateKey *rsa.PrivateKey, cfg *CertConfig) (*x509.CertificateRequest, error) {
	tmpl := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
			Country:      []string{"CN"},
		},
		DNSNames: cfg.DNSName,
	}
	certDERBytes, err := x509.CreateCertificateRequest(rand.Reader, &tmpl, privateKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificateRequest(certDERBytes)
}
