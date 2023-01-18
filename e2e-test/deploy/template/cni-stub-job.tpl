apiVersion: batch/v1
kind: Job
metadata:
  name: cni-{{ .host_name }}
  namespace: multi-nic-cni-operator-system
  labels:
    app: cni-stub
spec:
  template:
    metadata:
      labels:
        app: cni-stub
    spec:
      containers:
      - env:
        image: e2e-test/cni-stub:latest
        imagePullPolicy: IfNotPresent
        name: cni
        command: ["/bin/bash", "-c"]
        args: {{ .args }}
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: multi-nic-cni-operator-controller-manager
      serviceAccountName: multi-nic-cni-operator-controller-manager
      terminationGracePeriodSeconds: 30
  backoffLimit: 4