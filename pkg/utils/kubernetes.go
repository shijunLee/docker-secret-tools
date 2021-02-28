package utils

import (
	"io/ioutil"
	"os"
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
