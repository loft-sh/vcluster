# Testing the traefik CRD syncing

1. Once you've followed the installation instructions for setting up and installing Traefik on the host cluster and verified
that the installation works, proceed to the next step of creating the vCluster.

2. Create a vCluster with the above config as a values file

  ```bash
  vcluster create vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster/main/generic-sync-examples/traefik/config.yaml
  ```

3. Create a simple webserver deployment within the vCluster for testing Treafik ingress.

  ```bash
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webserver
spec:
  selector:
    matchLabels:
      app: webserver
  template:
    metadata:
      labels:
        app: webserver
    spec:
      containers:
      - name: webserver
        image: nginx:1.16.1
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: webserver-service
  labels:
    app: webserver
spec:
  ports:
  - port: 80
  selector:
    app: webserver
EOF
  ```

3. Now create a Traefik `IngressRoute` inside the vCluster which will give you access to the simple webserver deployment.

  ```bash
cat <<EOF | kubectl apply -f -
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: webserver-ingress
spec:
  entryPoints:
    - websecure
  routes:
    - kind: Rule
      match: Host(`webserver.localhost`)
      services:
        - name: webserver-service
          port: 80
  tls: {}  # If you specify a default SSL cert in the host Treafik configuration, it will be used by the vcluster IngressRoute.
EOF
  ```

4. Attempt to open `https://webserver.localhost` on the machine running the cluster and you should see a "Welcome to nginx!" page.
