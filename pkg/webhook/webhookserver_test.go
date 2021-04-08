package webhook

import (
	"encoding/json"
	"fmt"
	"testing"

	jsonpatch "github.com/evanphx/json-patch"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var testyaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx-test
  name: nginx-test
  namespace: test1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-test
  strategy: {}
  template:
    metadata:
      labels:
        app: nginx-test
    spec:
      containers:
      - image: docker.shijunlee.local/library/nginx:latest
        name: nginx
        imagePullPolicy: IfNotPresent
        resources: {}
status: {}
`

var targetYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx-test
  name: nginx-test
  namespace: test1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-test
  strategy: {}
  template:
    metadata:
      labels:
        app: nginx-test
    spec:
      imagePullSecrets:
      - name: tpaas-itg
      containers:
      - image: docker.shijunlee.local/library/nginx:latest
        name: nginx
        imagePullPolicy: IfNotPresent
        resources: {}
status: {}`

func Test_JsonPatch(t *testing.T) {
	// Let's create a merge patch from these two documents...
	original, _ := yaml.YAMLToJSON([]byte(testyaml))
	// original := []byte(`{"name": "John", "age": 24, "height": 3.21}`)
	// target := []byte(`{"name": "Jane", "age": 24}`)
	//target, _ := yaml.YAMLToJSON([]byte(targetYaml))
	// patch, err := jsonpatch.CreateMergePatch(original, target)
	// if err != nil {
	// 	panic(err)
	// }
	var testTemplate = getPodTemplate(original, "Deployment")
	if testTemplate == nil {
		t.Fatal()
		return
	}
	var newTemp = *testTemplate
	newTemp.ImagePullSecrets = append(testTemplate.ImagePullSecrets, corev1.LocalObjectReference{Name: "tpaas-itg"})
	originalPodData, err := json.Marshal(testTemplate)
	if err != nil {
		t.Fatal(err)
	}
	newJsonData, err := json.Marshal(&newTemp)
	if err != nil {
		t.Fatal(err)
	}
	patchData, err := jsonpatch.CreateMergePatch(originalPodData, newJsonData)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("patch document:   %s\n", string(patchData))
	patchJSON := []byte(`[
		{"op": "replace", "path": "/name", "value": "Jane"} 
	]`)

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		panic(err)
	}
	data, err := patch.Apply(original)
	// Now lets apply the patch against a different JSON document...
	if err != nil {
		t.Fatal(err)
	}
	// alternative := []byte(`{"name": "Tina", "age": 28, "height": 3.75}`)
	modifiedAlternative, _ := jsonpatch.MergePatch(original, patchData)

	fmt.Printf("patch document:   %s\n", string(data))
	fmt.Printf("updated alternative doc: %s\n", modifiedAlternative)
}
