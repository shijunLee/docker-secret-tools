module github.com/shijunLee/docker-secret-tools

go 1.14

require (
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/thedevsaddam/gojsonq v2.3.0+incompatible
	go.uber.org/zap v1.15.0
	gomodules.xyz/jsonpatch/v2 v2.1.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.2
	sigs.k8s.io/yaml v1.2.0
)
