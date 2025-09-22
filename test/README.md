### How to execute the e2e tests locally

1. Start test environment:
    - Start the vcluster in test mode `devspace run test <distro> <valuesFilePath from the testsuite>`
    
      - To run the test from the a general test suite i.e from `./test/e2e`.
      
          `devspace --namespace vcluster run test k3s`
      
      - To run tests from a specific test suite, you'll need to specify the values file path from that test suite.
      
          `devspace --namespace vcluster run test k3s --var VALUES_FILE=./test/e2e_node/values.yaml`
    
    - Then run following command in the terminal to start vcluster syncer.
        - To run default test suite start syncer with following command
        ```
        go run -mod vendor cmd/vcluster/main.go start --sync 'networkpolicies' --name=vcluster --service-account=vc-workload-vcluster --kube-config-context-name=my-vcluster --leader-elect=false --sync=nodes --sync=-ingressclasses --node-selector=kubernetes.io/hostname=kind-control-plane '--map-host-service=test/test=default/test' '--map-virtual-service=test/test=test'
        ```

        - To run tests from other test suites you'll need to change the flags for `go run -mod vendor cmd/vcluster/main.go start` accordingly. You can check the list of syncer flags by running `helm template vcluster ./charts/k3s/ -f ./test/commonValues.yaml -f ./test/<test_suite>/values.yaml`
        
         For e.g.
         ```
         helm template vcluster ./charts/k3s/ -f ./test/commonValues.yaml

         # Then look for `name: syncer` container

        - name: syncer
        image: "REPLACE_IMAGE_NAME"
        args:
          - --name=vcluster
          - --service-account=vc-workload-vcluster
          - --kube-config-context-name=my-vcluster
          - --leader-elect=false
          - --sync=nodes
          - --sync=-ingressclasses
          - --node-selector=kubernetes.io/hostname=kind-control-plane
          - "--target-namespace=vcluster-workload"
          - '--map-host-service=test/test=default/test'
          - '--map-virtual-service=test/test=test'
         ```
         The these flags shall be used with `go run -mod vendor cmd/vcluster/main.go start`
          
2. Then start the e2e tests via 
    ```
    cd test/<test_suite_path>
    VCLUSTER_NAMESPACE=vcluster go test -v -ginkgo.v -ginkgo.skip='.*NetworkPolicy.*'
    ```

