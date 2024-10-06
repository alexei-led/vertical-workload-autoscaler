#!/bin/bash

# Define the number of Redis nodes
NODES=3

# Loop through each node and perform the flush and reset
for i in $(seq 0 $((NODES - 1))); do
    echo "Resetting Redis node: redis-cluster-$i"
    
    # Flush all data
    kubectl exec redis-cluster-$i -- redis-cli flushall
    
    # Reset the cluster state
    kubectl exec redis-cluster-$i -- redis-cli cluster reset
done

# Collect the IP addresses of all nodes
NODE_IPS=$(kubectl get pods -l app=redis-cluster -o jsonpath='{range .items[*]}{.status.podIP}:6379 {end}')

# Recreate the Redis cluster
echo "Recreating Redis cluster with nodes: $NODE_IPS"
kubectl exec redis-cluster-0 -- redis-cli --cluster create $NODE_IPS --cluster-replicas 0 --cluster-yes

echo "Redis cluster has been reset and recreated."
