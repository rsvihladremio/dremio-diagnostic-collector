# running DDC from a kubernetes pod

One can run ddc directly from a Kubernetes pod assuming it's service account has been assigned at least this role

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ddc-collect
rules:
- apiGroups: [""] # core
  resources: ["pods", "pods/log"]
  verbs: ["get", "list"]
- apiGroups: [""]
  resources: ["pods/exec"]
  verbs: ["create"]
```

Optionally to get a total diagnostic output of the Kubernetes environment one will need the following

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ddc-collect
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  - pods
  - pods/log
  - persistentvolumeclaims
  - resoucesquotas
  - services
  - endpoints
  verbs:
  - get
  - list
- apiGroups:
  - events.k8s.io
  resources:
  - events
  verbs:
  - get
  - list
- apiGroups:
  - ""
  resources:
  - pods/exec
  verbs:
  - create
- apiGroups:
  - batch
  resources:
  - cronjobs
  - jobs
  verbs:
  - get
  - list
- apiGroups:
  - apps
  resources:
  - deployments
  - statefulsets
  - daemonsets
  - replicasets
  verbs:
  - get
  - list
- apiGroups:
  - networking.k8s.io
  resources:
  - ingresses
  verbs:
  - get
  - list
- apiGroups:
  - autoscaling
  resources:
  - horizontalpodautoscalers
  verbs:
  - get
  - list
- apiGroups:
  - policy
  resources:
  - poddisruptionbudgets
  verbs:
  - get
  - list
  ---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ddc-collect
rules:
- apiGroups:
  - ""
  resources:
  - nodes
  - persistentvolumes
  - limitranges
  - resoucesquotas
  - services
  - endpoints
  verbs:
  - get
  - list
- apiGroups:
  - storage.k8s.io
  resources:
  - storageclasses
  verbs:
  - get
  - list
- apiGroups:
  - scheduling.k8s.io
  resources:
  - priorityclasses
  verbs:
  - get
  - list
  ```

  Then a role binding would be need to be created for each type of role, for example in this case assuming we have a service account called ddc-collect the follow two bindings would need to be completed.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ddc-collect
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ddc-collect
subjects:
- kind: ServiceAccount
  name: ddc-collect
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ddc-collect
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ddc-collect
subjects:
- kind: ServiceAccount
  name: ddc-collect
  namespace: default
```
