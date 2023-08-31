# Testing the certmanager CRD syncing

1. Once you've followed the installation instructions for setting up and installing certmanager on the host cluster and verified
that the installation works, proceed to the next step of creating the vcluster
2. Create a vcluster with the above config as a values file

  ```bash
  vcluster create vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster/main/generic-sync-examples/cert-manager/config.yaml
  ```

3. Now try creating an `Issuer` and a self-signed `Certificate` inside the newly created vcluster

  ```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: cert-manager-test
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: test-selfsigned
  namespace: cert-manager-test
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: selfsigned-cert
  namespace: cert-manager-test
spec:
  dnsNames:
    - example.com
  secretName: selfsigned-cert-tls
  issuerRef:
    name: test-selfsigned
EOF
  ```
