---
name: Bug report
about: Create a report to help us improve
title: ''
labels: ''
assignees: ''

---

- [ ] checked https://foundation-model-stack.github.io/multi-nic-cni/user_guide/troubleshooting/.
- [ ] titled with the bug issue (if applicable).
- [ ] provided corresponding information regarding the troubleshooting guidelines (CR list/detail, multi-nic cni controller and/or daemon status/log).

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. 
2. 
3. 
4. 

**Expected behavior**
A clear and concise description of what you expected to happen.

**Screenshots**
If applicable, add screenshots to help explain your problem.
 - manager container of controller and multi-nicd DS status:
 - multinicnetwork CR:
 - hostinterface list/CR:
 - cidr CR (multiNICIPAM: true):
 - ippools CR (multiNICIPAM: true):
 - log of manager container: 
 - log of failed multi-nicd pod: 

**Environment (please complete the following information):**
 - platform: [e.g. self-managed k8s, self-managed OpenShift, EKS, IKS, AKS]
 - node profile:
 - operator version : 
 - cluster scale (number of nodes, pods, interfaces):

**Additional context**
Add any other context about the problem here.
