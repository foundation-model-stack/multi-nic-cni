# Upstream Development

## Release Steps

1. Make sure that `doc` branch is synced to `main` branch by pushing PR from `doc` to `main`.

2. Make sure that all workflows on the `main` branch are completed successfully.

3. Create a release on GitHub repository: https://github.com/foundation-model-stack/multi-nic-cni/releases
    * Create a new tag with format `release-vX.Y.Z` from the main branch. 
    * Add [auto-generate release note](https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes).
    * Insert summarized updates from [milestone](https://github.com/foundation-model-stack/multi-nic-cni/milestones).
4. Set new version with the following command and push a PR `upgrade version: X.Y.Z` to the main branch.
    
        make set_version X.Y.Z

5. Update catalog template file and push PR to community operator hub:
    * https://github.com/redhat-openshift-ecosystem/community-operators-prod
    * https://github.com/k8s-operatorhub/community-operators

6. Once the above PR merged, update release page in documentation. Check [documentation update guide](../contributing/local_build_push.md#documentation-update).
