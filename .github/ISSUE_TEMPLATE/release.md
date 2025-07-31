---
name: Release Checklist
about: Create a release checklist
title: 'Release v<X.Y.Z>'
labels: epic
assignees: ''

---


## Release Steps
(ref: https://foundation-model-stack.github.io/multi-nic-cni/contributing/maintainer/)

### 1. Tag release
- [ ] Make sure that doc branch is synced to main branch by pushing PR from doc to main.
- [ ] Make sure that all workflows on the main branch are completed successfully.
<!-- Attach the release tag link. -->
- [ ] Create a release on GitHub repository: https://github.com/foundation-model-stack/multi-nic-cni/releases

### 2. Increase version
<!-- Replace X.Y.Z+ with next version. 
For example, the next version of release 0.0.1 is 0.0.2, replace X.Y.Z+ here with 0.0.2. -->
- [ ] Prepare kbuilder image for new version. Dispatch the [workflow](https://github.com/foundation-model-stack/multi-nic-cni/actions/workflows/build_push_kbuilder.yaml) with image version X.Y.Z+.
<!-- Attach the PR link. -->
- [ ] Set new version with the following command and push a PR upgrade version: X.Y.Z+ to the main branch.

### 3. Upload to community operator hub
<!-- Attach the PR links to each item. -->
- [ ] Update catalog template file and push PR to community operator hub - [OpenShift](https://github.com/redhat-openshift-ecosystem/community-operators-prod)
- [ ] Update catalog template file and push PR to community operator hub - [OperatorHub.io](https://github.com/k8s-operatorhub/community-operators)
### 4. Update release document
- [ ] Update release page in documentation. Check [documentation update guide](https://foundation-model-stack.github.io/multi-nic-cni/contributing/local_build_push/#documentation-update)
