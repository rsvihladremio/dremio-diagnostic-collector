# Troubleshooting

## Missing Nodes or None Found

Make sure the labels you use actually correspond to kubernetes nodes in your pod. Run the following command to see your labels

```bash
 kubectl get pods --show-labels
NAME                READY   STATUS    RESTARTS   AGE     LABELS
dremio-executor-0   1/1     Running   0          6h32m   app=dremio-executor,controller-revision-hash=dremio-executor-cf89fc46b,role=dremio-cluster-pod,statefulset.kubernetes.io/pod-name=dremio-executor-0
dremio-executor-1   1/1     Running   0          6h32m   app=dremio-executor,controller-revision-hash=dremio-executor-cf89fc46b,role=dremio-cluster-pod,statefulset.kubernetes.io/pod-name=dremio-executor-1
dremio-master-0     1/1     Running   0          6h32m   app=dremio-coordinator,controller-revision-hash=dremio-master-7bd8d69c,role=dremio-cluster-pod,statefulset.kubernetes.io/pod-name=dremio-master-0
zk-0                1/1     Running   0          6h32m   app=zk,controller-revision-hash=zk-67ffd74b67,statefulset.kubernetes.io/pod-name=zk-0
```

pick one for executors and one for coordinators that matches what pods you want logs for. In this case since I want coordinators and executors I can run the following command

```bash
ddc -k -e app=dremio-executor -c app=dremio-coordinator
```

## No job profiles collected


You can grep for WARN logs and will be able to see something like this

```
WARN:  2023/06/06 17:35:18 local.go:823: no queries.json files found. This is probably an executor, so we are skipping collection of Job Profiles
```

Another problem could be our helm charts by default do not persist logs including queries.json (as of June 6th 2023). In the helm charts I suggest adding the following flag under `coordinator.extraStartParams`

```yaml
coordinator:
  extraStartParams: >-
    -Ddremio.log.path=/opt/dremio/data/logs
```

## No GC logs collected

Another problem could be our helm charts by default do not persist gc.logs  (as of June 6th 2023). In the helm charts I suggest adding the following flag under `coordinator.extraStartParams`

```yaml
coordinator:
  extraStartParams: >-
    -Xloggc:/opt/dremio/data/logs/gc.log
    -XX:+PrintGCDetails
    -XX:+PrintGCDateStamps
    -XX:+PrintTenuringDistribution
    -XX:+PrintGCCause
    -XX:+UseGCLogFileRotation
    -XX:NumberOfGCLogFiles=10
    -XX:GCLogFileSize=5M
```
