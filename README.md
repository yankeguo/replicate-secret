# replicate-secret

A tool for replicating Kubernetes Secrets

## Usage

```shell
/replicate-secret --source default/tls-cluster-wildcard --namespace '.*' --name tls-cluster
```

## Example

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: replicate-secret
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "create", "update"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["list"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: replicate-secret
automountServiceAccountToken: true
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: replicate-secret
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: replicate-secret
subjects:
  - kind: ServiceAccount
    name: replicate-secret
    namespace: kube-system
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: replicate-tls
spec:
  schedule: "H */6 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: replicate-secret
          containers:
            - name: replicate-tls
              image: yankeguo/replicate-secret:0.1.0
              command:
                [
                  "/replicate-secret",
                  "-source",
                  "tls-cluster-wildcard",
                  "-namespace",
                  ".*",
                ]
          restartPolicy: OnFailure
```

## Credits

GUO YANKE, MIT License
