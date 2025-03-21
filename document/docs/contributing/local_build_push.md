# Local Development
## Build images
### Build CNI operator
1. Set `IMAGE_REGISTRY` and `VERSION` environment to target image repository for operator

         export IMAGE_REGISTRY=<registry>
         export VERSION=<version>

2. For private image registry, follow these additional steps to add image-pulling secret
    * Put your secret for pulling operator image (`operator-secret.yaml`) to the secret folder

            mv operator-secret.yaml config/secret

    * Run script to update relevant kustomization files

            export OPERATOR_SECRET_NAME=$(cat config/secret/operator-secret.yaml|yq .metadata.name)
            make operator-secret

2. Build and push operator image

         go mod tidy
         make docker-build docker-push

3.  Build and push bundle image (optional)

         make bundle
         make bundle-build bundle-push

    To test the bundle, run

         operator-sdk run bundle ${IMAGE_REGISTRY}/multi-nic-cni-bundle:v${VERSION}

4. Build and use catalog source image (optional)

         make catalog-build catalog-push

    To test deploy with customized catalog, deploy the catalogsource resource.

         apiVersion: operators.coreos.com/v1alpha1
         kind: CatalogSource
         metadata:
           name: multi-nic-cni-operator
           namespace: openshift-marketplace
         spec:
           displayName: Multi-NIC CNI Operator
           publisher: IBM
           sourceType: grpc
           image: # <YOUR_CATALOG_IMAGE>
           updateStrategy:
             registryPoll:
               interval: 10m

     Then, you can deploy via console or subscription resource.

### Build CNI daemon
1. Set `IMAGE_REGISTRY` and `VERSION` environment to target image repository for daemon

         export IMAGE_REGISTRY=<registry>
         export VERSION=<version>

2. For private image registry, follow these additional steps to add image-pulling secret
    * Put your secret for pulling daemon image (`daemon-secret.yaml`) to the secret folder

            mv daemon-secret.yaml config/secret

    * Run script to update relevant kustomization files

            export DAEMON_SECRET_NAME=$(cat config/secret/daemon-secret.yaml|yq .metadata.name)
            make daemon-secret

3. Build and push daemon image

            # build environment: 
            #   Linux systems with netlink library
            cd daemon
            go mod tidy
            make docker-build-push

    This will also build the cni binary and copy the built binary to daemon component.

## Test
```bash
# test golang linter (available after v1.0.4)
make golint
# test controller 
make test
# test daemon (available after v1.0.3)
make test-daemon
```

## Install operator
```bash
make deploy
```

## Uninstall operator
```bash
make undeploy
```

## Documentation update
We build the document website using GitHub Page site which is based on [mkdocs](https://squidfunk.github.io/mkdocs-material/publishing-your-site/).

The material for building the website is under the folder `documentation`.
To test the updated document locally, please [install `mkdocs`](https://www.mkdocs.org/getting-started/) in your local environment and test with `mkdocs serve` under the folder `documentation`.

Please do not include any changes outside `documentation` folder and push the PR towards `doc` branch instead of `main` branch. 
