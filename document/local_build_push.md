# Locally Build and Deploy
## Build images
### 1. Build CNI operator
1. Set `IMAGE_REGISTRY` and `VERSION` environment to target image repository for operator
   ```bash
   export IMAGE_REGISTRY=<registry>
   export VERSION=<version>
   ```
2. For private image registry, follow these additional steps to add image-pulling secret
   1. Put your secret for pulling operator image (`operator-secret.yaml`) to the secret folder
        ```bash
        mv operator-secret.yaml config/secret
        ```
   2. Run script to update relevant kustomization files
      ```bash 
      export OPERATOR_SECRET_NAME=$(cat config/secret/operator-secret.yaml|yq .metadata.name)
      make operator-secret
      ```
3. Build and push operator image
    ```bash
    go mod tidy
    make docker-build docker-push
    ```
4.  Build and push bundle image (optional)
    ```bash
    make bundle
    make bundle-build bundle-push
    ```
    To test the bundle, run
    ```bash
    operator-sdk run bundle ${IMAGE_REGISTRY}/multi-nic-cni-bundle:v${VERSION}
    ```
### 2. Build CNI daemon
1. Set `IMAGE_REGISTRY` and `VERSION` environment to target image repository for daemon
   ```bash
   export IMAGE_REGISTRY=<registry>
   export VERSION=<version>
   ```
2. For private image registry, follow these additional steps to add image-pulling secret
   1. Put your secret for pulling daemon image (`daemon-secret.yaml`) to the secret folder
      ```bash
      mv daemon-secret.yaml config/secret
      ```
   2. Run script to update relevant kustomization files
      ```bash 
      export DAEMON_SECRET_NAME=$(cat config/secret/daemon-secret.yaml|yq .metadata.name)
      make daemon-secret
      ```
3. Build and push daemon image
    ```bash
    # build environment: 
    #   Linux systems with netlink library
    cd daemon
    go mod tidy
    make docker-build-push
    ```
    This will also build the cni binary and copy the built binary to daemon component.

## Install operator
 ```bash
 make deploy
 ```

 ## Uninstall operator
 ```bash
 make undeploy
 ```
