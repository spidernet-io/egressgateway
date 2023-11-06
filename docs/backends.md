# egressgateway

## Background

![egressgateway](./Egress-Gateway.png)

Starting with 2021, we received some feedback as follows.

There are two clusters A and B. Cluster A is VMWare-based and runs mainly Database workloads, and Cluster B is a Kubernetes cluster. Some applications in Cluster B need to access the database in Cluster A, and the network administrator wants the cluster Pods to be managed through an egress gateway.
