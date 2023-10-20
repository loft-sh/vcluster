## Load Testing for vcluster (Work In Progress)

How to use:

1. Create a new vcluster
2. Start testing via:
```
go run hack/load-testing/main.go -amount 1000 secrets
```

### Connect to SQLite

Install SQLite:
```
kubectl exec -it -n NAMESPACE NAME-0 -c syncer -- sh -c 'apk update && apk add sqlite'
```

Access Database:
```
kubectl exec -it -n NAMESPACE NAME-0 -c syncer -- sh -c 'sqlite3 /data/server/db/state.db'
```

