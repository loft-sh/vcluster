Create via:
```
kubectl create ns vcluster-test
kubectl apply -n vcluster-test -f hack/load-testing/deploy/k3s-mysql/mysql.yaml
vcluster create test -f hack/load-testing/deploy/k3s-mysql/values.yaml
```