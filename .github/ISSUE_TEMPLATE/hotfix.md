---
name: Hotfix Checklist
about: Create a hotfix checklist
title: 'Hotfix Issue <Issue Number> from Release v<X.Y.Z>'
labels: ["epic", "hotfix"]
assignees: ''

---
## Issue Link

- 

## Hotfix Injection Steps

- [ ] Fetch tags `git fetch --all --tags`

### Target Release v<X.Y.Z>
- [ ] Prepare hotfix branch 
<!---
    ```
    git checkout -b release-v[VERSION]-hotfix-<ISSUE> release-v[VERSION]
    ```
-->
- [ ] Commit add target branch on build workflow as needed 
<!---
    For example:

    ```diff
        1 change: 1 addition & 0 deletions 1  
        .github/workflows/build_push_daemon.yaml
 
        Original file line number	Diff line number	Diff line change
        @@ -4,6 +4,7 @@ on:
        push:
            branches:
            - main
    +       - release-v1.2.6-hotfix-278
    ```
-->
- [ ] Push prepared branch to mainstream
- [ ] Create hotfix PR to the prepared branch
- [ ] Clean the branch after merge

### Target Release v<X.Y.Z>
- [ ] Prepare hotfix branch 
- [ ] Commit add target branch on build workflow as needed 
- [ ] Push prepared branch to mainstream
- [ ] Create hotfix PR to the prepared branch
- [ ] Clean the branch after merge


### Main (if applicable)
- [ ] Create hotfix PR to the main branch
