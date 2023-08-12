# Tekton Pipelines

## Install tekton pipelines on host cluster

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

Optionally install tekton-dashboard on the host cluster

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/dashboard/latest/tekton-dashboard-release.yaml
```

## Deploy vcluster with tekton config

```bash
vcluster create vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster/main/generic-sync-examples/tekton-pipelines/config.yaml
```

## Create some tekton Tasks and Pipelines in vcluster

Some basic tasks

```bash
cat << EOF | kubectl apply -f -
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: hello
spec:
  params:
    - name: name
      type: string
  steps:
    - name: hello
      image: ubuntu
      command:
        - echo
      args:
        - "hello $(params.name)!"
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: greeting
spec:
  params:
    - name: name
      type: string
  steps:
    - name: greeting
      image: ubuntu
      command:
        - echo
      args:
        - "greetings $(params.name)!"
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: goodbye
spec:
  params:
    - name: name
      type: string
  steps:
    - name: goodbye
      image: ubuntu
      command:
        - echo
      args:
        - "goodbye $(params.name)!"
EOF
```

A pipeline that uses these tasks

```bash
cat << EOF | kubectl apply -f -
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: greetings
spec:
  params:
    - name: name
      type: string
      default: "anonymous"
  tasks:
    - name: hello
      taskRef:
        name: hello
      params: 
        - name: name
          value: "$(params.name)"
    - name: greeting
      taskRef:
        name: greeting
      params: 
        - name: name
          value: "$(params.name)"
    - name: goodbye
      taskRef:
        name: goodbye
      params: 
        - name: name
          value: "$(params.name)"
EOF
```

## Run the pipeline with a PipelineRun

```bash
cat << EOF | kubectl apply -f -
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: greetings-
spec:
  params:
    - name: name
      value: ishan
  pipelineRef:
    name: greetings
EOF
```
