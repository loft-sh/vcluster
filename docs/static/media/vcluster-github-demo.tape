# vhs tape for the vCluster README hero GIF.
#
# Requirements: Docker running + vcluster CLI >= 0.36 + vhs (brew install vhs).
# No Kubernetes cluster is needed. The tape uses the Docker driver (vind), so it
# is reproducible on any laptop or CI runner that has Docker.
#
# To regenerate, run these three steps from this directory:
#
#   vcluster create team-b --driver docker                     # 1. pre-step,
#   kubectl config use-context <your-own-context>               #    then switch back
#   vhs vcluster-github-demo.tape                              # 2. record
#   vcluster delete team-a --driver docker; \
#     vcluster delete team-b --driver docker                   # 3. cleanup
#
# Design: the gif shows one thing, the isolation boundary. Tenant team-a installs
# a CRD as cluster-admin; connect to tenant team-b and the same command returns
# nothing. Every displayed command and line of output is real (no mocks).
#
# Notes for whoever edits this next:
#   - Do not put `vcluster connect` in the tape. It hangs at "Waiting for vCluster
#     kubeconfig to be available" inside the vhs pty and kills the recording with
#     a bare "recording failed", even though the same command returns in under
#     three seconds in a normal shell. Switch tenants with `kubectl config
#     use-context` instead; the pre-step registers the context.
#   - Keep the recorded session short. Creating both tenants inside the tape and
#     tearing them down in Hide blocks makes for a long session; team-b is created
#     in the pre-step and teardown is a post-step for that reason.
#   - Wait+Screen matches the whole visible screen, not just new output. Match on
#     a string that cannot already be on screen (the context name, not "Switched
#     active kube context") or the wait returns instantly and the tape races
#     ahead of the cluster.
#   - The node needs roughly half a minute after create to report Ready. Without
#     that pause `kubectl get nodes` records a NotReady node.
#   - Deliberately no `vcluster --version` line. Printing a version number bakes a
#     release into a binary asset and dates the gif on every release. `kubectl get
#     nodes` surfaces the Kubernetes version instead and re-dates itself for free.

Output "vcluster-github-demo.gif"

Set FontSize 20
Set Width 1400
Set Height 820
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
Type "printf '%s\n' 'apiVersion: apiextensions.k8s.io/v1' 'kind: CustomResourceDefinition' 'metadata:' '  name: widgets.demo.io' 'spec:' '  group: demo.io' '  scope: Namespaced' '  names:' '    plural: widgets' '    kind: Widget' '  versions:' '  - name: v1' '    served: true' '    storage: true' '    schema:' '      openAPIV3Schema:' '        type: object' > widget-crd.yaml"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 300ms
Set TypingSpeed 45ms
Show

# A real Kubernetes cluster, in Docker, with no Kubernetes cluster underneath it.
Type "vcluster create team-a --driver docker"
Enter
Sleep 3s

Hide
Wait+Screen@600s /vcluster-docker_team-a/
Sleep 25s
Show
Sleep 2s

# It is a real cluster: its own node, its own Kubernetes version.
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

# Switch to the second tenant. A plain context switch: two tenants, two contexts,
# two clusters.
Type "kubectl config use-context vcluster-docker_team-b"
Enter
Sleep 2500ms

# The payoff: same command, different tenant, nothing there.
Type "kubectl get crd"
Enter
Sleep 5s
