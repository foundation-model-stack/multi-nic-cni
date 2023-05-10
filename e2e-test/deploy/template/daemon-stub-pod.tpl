apiVersion: v1
kind: Pod
metadata:
  name: multi-nicd-stub-{{ .index }}
spec:
  containers:
  - env:
    - name: DAEMON_PORT
      value: "11000"
    - name: HOST_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    - name: NODENAME
      value: kwok-node-{{ .index }}
    image: {{ .image }}
    imagePullPolicy: IfNotPresent
    name: multi-nicd
    ports:
    - containerPort: 11000
      protocol: TCP
    securityContext:
      privileged: true
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
  dnsPolicy: ClusterFirst
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: multi-nic-cni-operator-controller-manager
  serviceAccountName: multi-nic-cni-operator-controller-manager
  terminationGracePeriodSeconds: 30