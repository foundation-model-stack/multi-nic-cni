---
name: Release Checklist
about: Create a release checklist
title: 'Release v<X.Y.Z>'
labels: epic
assignees: ''

---

## Release Steps
(ref: https://foundation-model-stack.github.io/multi-nic-cni/contributing/maintainer/)

- [ ] Make sure that doc branch is synced to main branch by pushing PR from doc to main.
- [ ] Make sure that all workflows on the main branch are completed successfully.
- [ ] Create a release on GitHub repository: https://github.com/foundation-model-stack/multi-nic-cni/releases
- [ ] Prepare kbuilder image for new version. Dispatch the [workflow](https://github.com/foundation-model-stack/multi-nic-cni/actions/workflows/build_push_kbuilder.yaml) with image version X.Y.Z.
- [ ] Set new version with the following command and push a PR upgrade version: X.Y.Z to the main branch.
- [ ] Update catalog template file and push PR to community operator hub - [OpenShift](https://github.com/redhat-openshift-ecosystem/community-operators-prod)
- [ ] Update catalog template file and push PR to community operator hub - [OperatorHub.io](https://github.com/k8s-operatorhub/community-operators)
- [ ] Update release page in documentation. Check [documentation update guide](https://foundation-model-stack.github.io/multi-nic-cni/contributing/local_build_push/#documentation-update)
