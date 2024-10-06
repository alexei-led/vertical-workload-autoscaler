# Setup Cluster

```bash
kubectl apply -f redis-configmap.yaml
kubectl apply -f redis-statefulset.yaml
kubectl apply -f redis-service.yaml
```

Enable cluster mode by running the following command. Make sure to use the DNS names of the Redis headless services to create the cluster. This will allow the Redis nodes to communicate with each other using the DNS names of the Redis headless services. And also, recover from Pod restarts/upgrades.

```shell
# Create the Redis Cluster using the DNS names of the Redis headless services 
kubectl exec -it redis-cluster-0 -- redis-cli --cluster create $(kubectl get pods -l app=redis-cluster -o jsonpath='{range .items[*]}{.metadata.name}.redis-cluster.default.svc.cluster.local:6379 {end}') --cluster-replicas 0
        
```

# Verify Cluster

```shell
kubectl get statefulset
kubectl get pods
kubectl exec -it redis-cluster-0 -- redis-cli cluster nodes
kubectl exec -it redis-cluster-0 -- redis-cli cluster info
```

Run benchmark (in loop):

```bash
kubectl run redis-client --rm -it --image  redis:6.2 -- bash

cat << EOF > run_benchmark.sh
#!/bin/bash

while true; do
  echo "Starting benchmark run at $(date)"
  redis-benchmark -h redis-cluster-0.redis-cluster -p 6379 -t set,get -n 10000000 -d 100 -r 100000 -P 50 --cluster
  echo "Benchmark run completed at $(date)"
  echo "Sleeping for 10 seconds..."
  sleep 10
done
EOF

chmod +x run_benchmark.sh

./run_benchmark.sh
```

This benchmark will:
– Perform 10 million operations (5 million SETs and 5 million GETs)
– Use 100-byte payloads
– Distribute operations across 100,000 different keys
– Use pipelining for improved performance
– Operate correctly with your Redis Cluster setup

# Cleanup

```shell
kubectl delete -f redis-statefulset.yaml
kubectl delete pvc -l app=redis-cluster
kubectl delete -f redis-service.yaml
kubectl delete -f redis-configmap.yaml
kubectl delete -f redis-vpa.yaml
kubectl delete -f redis-vwa.yaml
```
