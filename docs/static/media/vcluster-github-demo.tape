# vhs tape for the vCluster README hero GIF.
# To regenerate: cd docs/static/media && vhs vcluster-github-demo.tape
# Requirements: Docker running (for background proxy) + vcluster CLI >= 0.33.1 + vhs (brew install vhs).
# Needs a reachable Kubernetes cluster. The tape honours $KUBECONFIG from the
# caller's environment — export KUBECONFIG=/path/to/your/kubeconfig before running.
#
# Design: every displayed command and line of output is real output from the
# CLI (no mocks). Hide blocks skip the boring "Waiting for pod / Init:0/1"
# progress lines so the gif shows only user-interesting moments.

Output "vcluster-github-demo.gif"

Set FontSize 20
Set Width 1400
Set Height 760
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set Padding 30
Set Shell "zsh"

# Reset CLI default driver to helm + clean slate (hidden from gif)
Hide
Type "vcluster use driver helm"
Enter
Sleep 800ms
Type "clear"
Enter
Sleep 300ms
Show

Type "vcluster --version"
Enter
Sleep 1500ms

# Matches the README Quick Start exactly
Type "vcluster create my-vcluster --namespace team-x"
Enter
Sleep 3500ms

Hide
Wait+Screen@240s /Switched active kube context/
Show
Sleep 2500ms

# Inside the Tenant Cluster — real kubectl output
Type "kubectl get namespaces"
Enter
Sleep 4s

Type "vcluster delete my-vcluster --namespace team-x"
Enter
Sleep 2500ms

# Wait for the TRUE final marker so the gif ends on a clean state
Hide
Wait+Screen@120s /Virtual Cluster is deleted/
Show
Sleep 2500ms
