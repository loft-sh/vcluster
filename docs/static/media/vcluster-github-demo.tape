# vhs tape for the vCluster README hero GIF.
#
# Requirements: vhs (brew install vhs), the vcluster CLI >= 0.36, and a
# Kubernetes cluster to act as the Control Plane Cluster. The tape uses the
# default helm driver against a real cluster, which is the production path and
# the same command the README quickstart shows.
#
# To regenerate, run these steps. Any Kubernetes cluster works as the Control
# Plane Cluster; kind is used here so the recording is reproducible anywhere.
#
#   1. Control Plane Cluster and the second tenant:
#        kind create cluster --name vcluster-demo
#        cd $(mktemp -d)                   # the CLI refuses to run from a
#                                          # directory containing "vcluster"
#        vcluster create team-b --namespace team-b --driver helm
#        kubectl config use-context kind-vcluster-demo
#
#   2. Record:
#        vhs vcluster-github-demo.tape
#
#   NOTE: the tape sets the CLI default driver to helm so the recorded command
#   matches the README quickstart exactly (no --driver flag on screen). That is a
#   persisted setting. If your default was something else, restore it afterwards
#   with e.g. `vcluster use driver docker`.
#
#   3. Clean up:
#        vcluster delete team-a --namespace team-a --driver helm
#        vcluster delete team-b --namespace team-b --driver helm
#        kind delete cluster --name vcluster-demo
#
# Design: the gif shows one thing, the isolation boundary. Tenant team-a installs
# a CRD as cluster-admin; switch to sibling tenant team-b and the same command
# returns nothing. Every displayed command and line of output is real (no mocks).
#
# Notes for whoever edits this next:
#   - Set the driver explicitly in the hidden block. A bare `vcluster create`
#     uses whatever driver is persisted in ~/.vcluster/config.json, so on a
#     machine defaulted to docker the tape silently builds a docker cluster, the
#     Wait+Screen never matches, and the recording burns its full timeout.
#   - Create team-b from the Control Plane Cluster context, never from inside
#     team-a. Creating it while connected to team-a nests it inside team-a and
#     you get a chained context name, which is a different architecture and a
#     different story than two sibling tenants.
#   - Do not put `vcluster connect` or `vcluster disconnect` in the tape. They
#     hang on "Starting background proxy container" inside the vhs pty and kill
#     the recording with a bare "recording failed", even though the same command
#     returns in seconds in a normal shell. Switch with `kubectl config
#     use-context` instead.
#   - Wait+Screen matches the whole visible screen, not just new output. Match on
#     a string that cannot already be on screen (the full context name, not
#     "Switched active kube context") or the wait returns instantly and the tape
#     races ahead of the cluster.
#   - The node needs a few seconds after create to report Ready. Without the
#     pause `kubectl get nodes` records a NotReady node.
#   - Deliberately no `vcluster --version` line. Printing a version number bakes a
#     release into a binary asset and dates the gif on every release. `kubectl get
#     nodes` surfaces the Kubernetes version instead and re-dates itself for free.

Output "vcluster-github-demo.gif"

Set FontSize 20
Set Width 1400
Set Height 680
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set Padding 30
Set Shell "zsh"

# Hidden setup: scratch dir, the demo CRD, and a clean slate.
Hide
Set TypingSpeed 1ms
Type "cd $(mktemp -d)"
Enter
Sleep 300ms
Type "vcluster use driver helm"
Enter
Sleep 800ms
Type "printf '%s\n' 'apiVersion: apiextensions.k8s.io/v1' 'kind: CustomResourceDefinition' 'metadata:' '  name: widgets.demo.io' 'spec:' '  group: demo.io' '  scope: Namespaced' '  names:' '    plural: widgets' '    kind: Widget' '  versions:' '  - name: v1' '    served: true' '    storage: true' '    schema:' '      openAPIV3Schema:' '        type: object' > widget-crd.yaml"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 300ms
Set TypingSpeed 45ms
Show

# A tenant cluster on a real Kubernetes cluster. Same command as the quickstart.
Type "vcluster create team-a --namespace team-a"
Enter
Sleep 3s

Hide
Wait+Screen@600s /vcluster_team-a_team-a_kind-vcluster-demo/
Sleep 20s
Show
Sleep 2s

# It is a real cluster, running the current Kubernetes.
Type "kubectl get nodes"
Enter
Sleep 3s

# The tenant is cluster-admin inside their own cluster.
Type "kubectl apply -f widget-crd.yaml"
Enter
Sleep 2500ms

Type "kubectl get crd"
Enter
Sleep 3500ms

# Switch to the sibling tenant on the same Control Plane Cluster.
Type "kubectl config use-context vcluster_team-b_team-b_kind-vcluster-demo"
Enter
Sleep 2500ms

# The payoff: same command, different tenant, nothing there.
Type "kubectl get crd"
Enter
Sleep 5s
