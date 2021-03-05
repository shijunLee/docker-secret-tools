package utils

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenerateCert(t *testing.T) {
	var config = &CertConfig{
		CertName:     "test",
		CertType:     ServingCert,
		CommonName:   "test.test.svc",
		Organization: []string{"test"},
		DNSName:      []string{"test.test.svc", "test", "test.test"},
	}
	if err := verifyConfig(config); err != nil {
		t.Fatal(err)
	}
	// If no custom CAKey and CACert are provided we have to generate them
	caKey, err := newPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pk, ca, cer, err := GenerateCert(config)
	if err != nil {
		t.Fatal(err)
	}
	dataPK := EncodePrivateKeyPEM(pk)
	fmt.Println(string(dataPK))
	dataCa := EncodeCertificatePEM(ca)
	fmt.Println(string(dataCa))
	dataCer := EncodeCertificatePEM(cer)
	fmt.Println(string(dataCer))

	request, err := CreateCertificateTool(caKey, config)
	if err != nil {
		t.Fatal(err)
	}
	dataRequest := EncodeCertificateRequestPEM(request)
	fmt.Println(string(dataRequest))
}

func TestInitCommand(t *testing.T) {

	hostDomain := fmt.Sprintf("${HOSTNAME}.%s.%s.svc.cluster.local", "test", "consul-test")
	nslookupCommand := `TIMEOUT_READY=0
			while ( ! nslookup {{HOSTS}} )
			do
			# If TIMEOUT_READY is 0 we should never time out and exit
			TIMEOUT_READY=$(( TIMEOUT_READY-1 ))
			if [ $TIMEOUT_READY -eq 0 ];then
			echo \"Timed out waiting for DNS entry\"
			exit 1
			fi
			sleep 1
			done`
	nslookupCommand = strings.ReplaceAll(nslookupCommand, "{{HOSTS}}", hostDomain)
	fmt.Print(nslookupCommand)
}
