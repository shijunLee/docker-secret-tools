package controller

import (
	"context"
	"testing"

	"github.com/shijunLee/docker-secret-tools/pkg/utils"
)

func Test_WorkloadJsonQ(t *testing.T) {

	var deploymentString = `
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app.kubernetes.io/component":"controller","app.kubernetes.io/instance":"ingress-nginx","app.kubernetes.io/managed-by":"Helm","app.kubernetes.io/name":"ingress-nginx","app.kubernetes.io/version":"0.44.0","helm.sh/chart":"ingress-nginx-3.23.0"},"name":"ingress-nginx-controller","namespace":"ingress-nginx"},"spec":{"minReadySeconds":0,"revisionHistoryLimit":10,"selector":{"matchLabels":{"app.kubernetes.io/component":"controller","app.kubernetes.io/instance":"ingress-nginx","app.kubernetes.io/name":"ingress-nginx"}},"template":{"metadata":{"labels":{"app.kubernetes.io/component":"controller","app.kubernetes.io/instance":"ingress-nginx","app.kubernetes.io/name":"ingress-nginx"}},"spec":{"containers":[{"args":["/nginx-ingress-controller","--publish-service=$(POD_NAMESPACE)/ingress-nginx-controller","--election-id=ingress-controller-leader","--ingress-class=nginx","--configmap=$(POD_NAMESPACE)/ingress-nginx-controller","--validating-webhook=:8443","--validating-webhook-certificate=/usr/local/certificates/cert","--validating-webhook-key=/usr/local/certificates/key"],"env":[{"name":"POD_NAME","valueFrom":{"fieldRef":{"fieldPath":"metadata.name"}}},{"name":"POD_NAMESPACE","valueFrom":{"fieldRef":{"fieldPath":"metadata.namespace"}}},{"name":"LD_PRELOAD","value":"/usr/local/lib/libmimalloc.so"}],"image":"k8s.gcr.io/ingress-nginx/controller:v0.44.0@sha256:3dd0fac48073beaca2d67a78c746c7593f9c575168a17139a9955a82c63c4b9a","imagePullPolicy":"IfNotPresent","lifecycle":{"preStop":{"exec":{"command":["/wait-shutdown"]}}},"livenessProbe":{"failureThreshold":5,"httpGet":{"path":"/healthz","port":10254,"scheme":"HTTP"},"initialDelaySeconds":10,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1},"name":"controller","ports":[{"containerPort":80,"name":"http","protocol":"TCP"},{"containerPort":443,"name":"https","protocol":"TCP"},{"containerPort":8443,"name":"webhook","protocol":"TCP"}],"readinessProbe":{"failureThreshold":3,"httpGet":{"path":"/healthz","port":10254,"scheme":"HTTP"},"initialDelaySeconds":10,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1},"resources":{"requests":{"cpu":"100m","memory":"90Mi"}},"securityContext":{"allowPrivilegeEscalation":true,"capabilities":{"add":["NET_BIND_SERVICE"],"drop":["ALL"]},"runAsUser":101},"volumeMounts":[{"mountPath":"/usr/local/certificates/","name":"webhook-cert","readOnly":true}]}],"dnsPolicy":"ClusterFirst","nodeSelector":{"kubernetes.io/os":"linux"},"serviceAccountName":"ingress-nginx","terminationGracePeriodSeconds":300,"volumes":[{"name":"webhook-cert","secret":{"secretName":"ingress-nginx-admission"}}]}}}}
  creationTimestamp: "2021-02-21T04:29:33Z"
  generation: 2
  labels:
    app.kubernetes.io/component: controller
    app.kubernetes.io/instance: ingress-nginx
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: ingress-nginx
    app.kubernetes.io/version: 0.44.0
    helm.sh/chart: ingress-nginx-3.23.0
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
        f:labels:
          .: {}
          f:app.kubernetes.io/component: {}
          f:app.kubernetes.io/instance: {}
          f:app.kubernetes.io/managed-by: {}
          f:app.kubernetes.io/name: {}
          f:app.kubernetes.io/version: {}
          f:helm.sh/chart: {}
      f:spec:
        f:progressDeadlineSeconds: {}
        f:replicas: {}
        f:revisionHistoryLimit: {}
        f:selector: {}
        f:strategy:
          f:rollingUpdate:
            .: {}
            f:maxSurge: {}
            f:maxUnavailable: {}
          f:type: {}
        f:template:
          f:metadata:
            f:labels:
              .: {}
              f:app.kubernetes.io/component: {}
              f:app.kubernetes.io/instance: {}
              f:app.kubernetes.io/name: {}
          f:spec:
            f:containers:
              k:{"name":"controller"}:
                .: {}
                f:args: {}
                f:env:
                  .: {}
                  k:{"name":"LD_PRELOAD"}:
                    .: {}
                    f:name: {}
                    f:value: {}
                  k:{"name":"POD_NAME"}:
                    .: {}
                    f:name: {}
                    f:valueFrom:
                      .: {}
                      f:fieldRef:
                        .: {}
                        f:apiVersion: {}
                        f:fieldPath: {}
                  k:{"name":"POD_NAMESPACE"}:
                    .: {}
                    f:name: {}
                    f:valueFrom:
                      .: {}
                      f:fieldRef:
                        .: {}
                        f:apiVersion: {}
                        f:fieldPath: {}
                f:imagePullPolicy: {}
                f:lifecycle:
                  .: {}
                  f:preStop:
                    .: {}
                    f:exec:
                      .: {}
                      f:command: {}
                f:livenessProbe:
                  .: {}
                  f:failureThreshold: {}
                  f:httpGet:
                    .: {}
                    f:path: {}
                    f:port: {}
                    f:scheme: {}
                  f:initialDelaySeconds: {}
                  f:periodSeconds: {}
                  f:successThreshold: {}
                  f:timeoutSeconds: {}
                f:name: {}
                f:ports:
                  .: {}
                  k:{"containerPort":80,"protocol":"TCP"}:
                    .: {}
                    f:containerPort: {}
                    f:name: {}
                    f:protocol: {}
                  k:{"containerPort":443,"protocol":"TCP"}:
                    .: {}
                    f:containerPort: {}
                    f:name: {}
                    f:protocol: {}
                  k:{"containerPort":8443,"protocol":"TCP"}:
                    .: {}
                    f:containerPort: {}
                    f:name: {}
                    f:protocol: {}
                f:readinessProbe:
                  .: {}
                  f:failureThreshold: {}
                  f:httpGet:
                    .: {}
                    f:path: {}
                    f:port: {}
                    f:scheme: {}
                  f:initialDelaySeconds: {}
                  f:periodSeconds: {}
                  f:successThreshold: {}
                  f:timeoutSeconds: {}
                f:resources:
                  .: {}
                  f:requests:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                f:securityContext:
                  .: {}
                  f:allowPrivilegeEscalation: {}
                  f:capabilities:
                    .: {}
                    f:add: {}
                    f:drop: {}
                  f:runAsUser: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
                f:volumeMounts:
                  .: {}
                  k:{"mountPath":"/usr/local/certificates/"}:
                    .: {}
                    f:mountPath: {}
                    f:name: {}
                    f:readOnly: {}
            f:dnsPolicy: {}
            f:nodeSelector:
              .: {}
              f:kubernetes.io/os: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:serviceAccount: {}
            f:serviceAccountName: {}
            f:terminationGracePeriodSeconds: {}
            f:volumes:
              .: {}
              k:{"name":"webhook-cert"}:
                .: {}
                f:name: {}
                f:secret:
                  .: {}
                  f:defaultMode: {}
                  f:secretName: {}
    manager: kubectl-client-side-apply
    operation: Update
    time: "2021-02-21T04:29:33Z"
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:template:
          f:spec:
            f:containers:
              k:{"name":"controller"}:
                f:image: {}
    manager: kubectl-edit
    operation: Update
    time: "2021-02-21T06:57:21Z"
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          f:deployment.kubernetes.io/revision: {}
      f:status:
        f:availableReplicas: {}
        f:conditions:
          .: {}
          k:{"type":"Available"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
          k:{"type":"Progressing"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
        f:observedGeneration: {}
        f:readyReplicas: {}
        f:replicas: {}
        f:updatedReplicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-02-27T15:07:32Z"
  name: ingress-nginx-controller
  namespace: ingress-nginx
  resourceVersion: "169873"
  uid: 82c2c97d-b74b-4bea-bb88-2d0d59261cf8
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app.kubernetes.io/component: controller
      app.kubernetes.io/instance: ingress-nginx
      app.kubernetes.io/name: ingress-nginx
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app.kubernetes.io/component: controller
        app.kubernetes.io/instance: ingress-nginx
        app.kubernetes.io/name: ingress-nginx
    spec:
      containers:
      - args:
        - /nginx-ingress-controller
        - --publish-service=$(POD_NAMESPACE)/ingress-nginx-controller
        - --election-id=ingress-controller-leader
        - --ingress-class=nginx
        - --configmap=$(POD_NAMESPACE)/ingress-nginx-controller
        - --validating-webhook=:8443
        - --validating-webhook-certificate=/usr/local/certificates/cert
        - --validating-webhook-key=/usr/local/certificates/key
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
        - name: LD_PRELOAD
          value: /usr/local/lib/libmimalloc.so
        image: docker.shijunlee.local/ingress-nginx/controller:v0.44.0
        imagePullPolicy: IfNotPresent
        lifecycle:
          preStop:
            exec:
              command:
              - /wait-shutdown
        livenessProbe:
          failureThreshold: 5
          httpGet:
            path: /healthz
            port: 10254
            scheme: HTTP
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: controller
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
        - containerPort: 443
          name: https
          protocol: TCP
        - containerPort: 8443
          name: webhook
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 10254
            scheme: HTTP
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        resources:
          requests:
            cpu: 100m
            memory: 90Mi
        securityContext:
          allowPrivilegeEscalation: true
          capabilities:
            add:
            - NET_BIND_SERVICE
            drop:
            - ALL
          runAsUser: 101
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /usr/local/certificates/
          name: webhook-cert
          readOnly: true
      dnsPolicy: ClusterFirst
      nodeSelector:
        kubernetes.io/os: linux
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: ingress-nginx
      serviceAccountName: ingress-nginx
      terminationGracePeriodSeconds: 300
      volumes:
      - name: webhook-cert
        secret:
          defaultMode: 420
          secretName: ingress-nginx-admission
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2021-02-21T06:57:21Z"
    lastUpdateTime: "2021-02-21T06:57:39Z"
    message: ReplicaSet "ingress-nginx-controller-968bffbf5" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  - lastTransitionTime: "2021-02-27T15:07:32Z"
    lastUpdateTime: "2021-02-27T15:07:32Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  observedGeneration: 2
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1`
	var podString = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    cni.projectcalico.org/podIP: 10.244.249.28/32
    cni.projectcalico.org/podIPs: 10.244.249.28/32
  creationTimestamp: "2021-02-21T06:57:21Z"
  generateName: ingress-nginx-controller-968bffbf5-
  labels:
    app.kubernetes.io/component: controller
    app.kubernetes.io/instance: ingress-nginx
    app.kubernetes.io/name: ingress-nginx
    pod-template-hash: 968bffbf5
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:generateName: {}
        f:labels:
          .: {}
          f:app.kubernetes.io/component: {}
          f:app.kubernetes.io/instance: {}
          f:app.kubernetes.io/name: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"e9f5e571-d6c5-4096-b91d-ceed82b5bbf2"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:containers:
          k:{"name":"controller"}:
            .: {}
            f:args: {}
            f:env:
              .: {}
              k:{"name":"LD_PRELOAD"}:
                .: {}
                f:name: {}
                f:value: {}
              k:{"name":"POD_NAME"}:
                .: {}
                f:name: {}
                f:valueFrom:
                  .: {}
                  f:fieldRef:
                    .: {}
                    f:apiVersion: {}
                    f:fieldPath: {}
              k:{"name":"POD_NAMESPACE"}:
                .: {}
                f:name: {}
                f:valueFrom:
                  .: {}
                  f:fieldRef:
                    .: {}
                    f:apiVersion: {}
                    f:fieldPath: {}
            f:image: {}
            f:imagePullPolicy: {}
            f:lifecycle:
              .: {}
              f:preStop:
                .: {}
                f:exec:
                  .: {}
                  f:command: {}
            f:livenessProbe:
              .: {}
              f:failureThreshold: {}
              f:httpGet:
                .: {}
                f:path: {}
                f:port: {}
                f:scheme: {}
              f:initialDelaySeconds: {}
              f:periodSeconds: {}
              f:successThreshold: {}
              f:timeoutSeconds: {}
            f:name: {}
            f:ports:
              .: {}
              k:{"containerPort":80,"protocol":"TCP"}:
                .: {}
                f:containerPort: {}
                f:name: {}
                f:protocol: {}
              k:{"containerPort":443,"protocol":"TCP"}:
                .: {}
                f:containerPort: {}
                f:name: {}
                f:protocol: {}
              k:{"containerPort":8443,"protocol":"TCP"}:
                .: {}
                f:containerPort: {}
                f:name: {}
                f:protocol: {}
            f:readinessProbe:
              .: {}
              f:failureThreshold: {}
              f:httpGet:
                .: {}
                f:path: {}
                f:port: {}
                f:scheme: {}
              f:initialDelaySeconds: {}
              f:periodSeconds: {}
              f:successThreshold: {}
              f:timeoutSeconds: {}
            f:resources:
              .: {}
              f:requests:
                .: {}
                f:cpu: {}
                f:memory: {}
            f:securityContext:
              .: {}
              f:allowPrivilegeEscalation: {}
              f:capabilities:
                .: {}
                f:add: {}
                f:drop: {}
              f:runAsUser: {}
            f:terminationMessagePath: {}
            f:terminationMessagePolicy: {}
            f:volumeMounts:
              .: {}
              k:{"mountPath":"/usr/local/certificates/"}:
                .: {}
                f:mountPath: {}
                f:name: {}
                f:readOnly: {}
        f:dnsPolicy: {}
        f:enableServiceLinks: {}
        f:nodeSelector:
          .: {}
          f:kubernetes.io/os: {}
        f:restartPolicy: {}
        f:schedulerName: {}
        f:securityContext: {}
        f:serviceAccount: {}
        f:serviceAccountName: {}
        f:terminationGracePeriodSeconds: {}
        f:volumes:
          .: {}
          k:{"name":"webhook-cert"}:
            .: {}
            f:name: {}
            f:secret:
              .: {}
              f:defaultMode: {}
              f:secretName: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-02-21T06:57:21Z"
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:cni.projectcalico.org/podIP: {}
          f:cni.projectcalico.org/podIPs: {}
    manager: calico
    operation: Update
    time: "2021-02-21T06:57:22Z"
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        f:conditions:
          k:{"type":"ContainersReady"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Initialized"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Ready"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
        f:containerStatuses: {}
        f:hostIP: {}
        f:phase: {}
        f:podIP: {}
        f:podIPs:
          .: {}
          k:{"ip":"10.244.249.28"}:
            .: {}
            f:ip: {}
        f:startTime: {}
    manager: kubelet
    operation: Update
    time: "2021-02-27T15:07:32Z"
  name: ingress-nginx-controller-968bffbf5-k4lz6
  namespace: ingress-nginx
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: ingress-nginx-controller-968bffbf5
    uid: e9f5e571-d6c5-4096-b91d-ceed82b5bbf2
  resourceVersion: "169867"
  uid: caae403c-0997-40a7-807e-36caf8e7da31
spec:
  containers:
  - args:
    - /nginx-ingress-controller
    - --publish-service=$(POD_NAMESPACE)/ingress-nginx-controller
    - --election-id=ingress-controller-leader
    - --ingress-class=nginx
    - --configmap=$(POD_NAMESPACE)/ingress-nginx-controller
    - --validating-webhook=:8443
    - --validating-webhook-certificate=/usr/local/certificates/cert
    - --validating-webhook-key=/usr/local/certificates/key
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          apiVersion: v1
          fieldPath: metadata.name
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          apiVersion: v1
          fieldPath: metadata.namespace
    - name: LD_PRELOAD
      value: /usr/local/lib/libmimalloc.so
    image: docker.shijunlee.local/ingress-nginx/controller:v0.44.0
    imagePullPolicy: IfNotPresent
    lifecycle:
      preStop:
        exec:
          command:
          - /wait-shutdown
    livenessProbe:
      failureThreshold: 5
      httpGet:
        path: /healthz
        port: 10254
        scheme: HTTP
      initialDelaySeconds: 10
      periodSeconds: 10
      successThreshold: 1
      timeoutSeconds: 1
    name: controller
    ports:
    - containerPort: 80
      name: http
      protocol: TCP
    - containerPort: 443
      name: https
      protocol: TCP
    - containerPort: 8443
      name: webhook
      protocol: TCP
    readinessProbe:
      failureThreshold: 3
      httpGet:
        path: /healthz
        port: 10254
        scheme: HTTP
      initialDelaySeconds: 10
      periodSeconds: 10
      successThreshold: 1
      timeoutSeconds: 1
    resources:
      requests:
        cpu: 100m
        memory: 90Mi
    securityContext:
      allowPrivilegeEscalation: true
      capabilities:
        add:
        - NET_BIND_SERVICE
        drop:
        - ALL
      runAsUser: 101
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /usr/local/certificates/
      name: webhook-cert
      readOnly: true
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: ingress-nginx-token-x5frb
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: k8snode1
  nodeSelector:
    kubernetes.io/os: linux
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: ingress-nginx
  serviceAccountName: ingress-nginx
  terminationGracePeriodSeconds: 300
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: webhook-cert
    secret:
      defaultMode: 420
      secretName: ingress-nginx-admission
  - name: ingress-nginx-token-x5frb
    secret:
      defaultMode: 420
      secretName: ingress-nginx-token-x5frb
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-02-21T06:57:21Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2021-02-27T15:07:32Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2021-02-27T15:07:32Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2021-02-21T06:57:21Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: docker://f659a1b36c55aec57d06ae0a33e7bc41f9199f4bac90945251036c1c42b9bf6e
    image: docker.shijunlee.local/ingress-nginx/controller:v0.44.0
    imageID: docker-pullable://docker.shijunlee.local/ingress-nginx/controller@sha256:d3e445ddbfa85165a152be1b1671580eb643c127cff8494c1243cd1330818efa
    lastState:
      terminated:
        containerID: docker://49aab0106a388ed2b0dec808bb5207b9ec6f43164333cb7a814263e2628cb2e8
        exitCode: 255
        finishedAt: "2021-02-27T15:05:13Z"
        reason: Error
        startedAt: "2021-02-23T15:38:41Z"
    name: controller
    ready: true
    restartCount: 3
    started: true
    state:
      running:
        startedAt: "2021-02-27T15:06:55Z"
  hostIP: 192.168.71.5
  phase: Running
  podIP: 10.244.249.28
  podIPs:
  - ip: 10.244.249.28
  qosClass: Burstable
  startTime: "2021-02-21T06:57:21Z"`
	t.Run("deploy", func(t *testing.T) {
		result, err := utils.GetImageFromYaml(context.TODO(), deploymentString)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(result)
	})
	t.Run("pod", func(t *testing.T) {
		result, err :=utils.GetImageFromYaml(context.TODO(), podString)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(result)
	})
}
