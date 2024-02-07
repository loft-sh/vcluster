Add [unittest plugin](https://github.com/helm-unittest/helm-unittest) via:
```
helm plugin install https://github.com/helm-unittest/helm-unittest.git
```

Run tests via:
```
helm unittest charts/eks -d
```
