# Contributor Guide

## Onboarding Multi-NIC CNI

1. Learn about Multi-NIC CNI. Recommend the blog post [Multi-NIC CNI Operator 101: Deep Dive into Container Multi-NIC Simplification](https://medium.com/@sunyanan.choochotkaew1/multi-nic-cni-operator-101-deep-dive-into-container-multi-nic-simplification-1b0cf5f67bb5). 
2. Deep dive into [Multi-NIC CNI architecture](./architecture.md) and [communication flow](../concept/multi-nic-ipam.md#workflows).
3. Prepare Cluster with required environments. Check [the requirements](../user_guide/index.md#requirements).
4. Install Multi-NIC CNI with common MultiNicNetwork definition. Check [installation guide](../user_guide/index.md#installation).
5. Test network connection. Check [user guide](../user_guide/user.md).
6. Check [development guide](./local_build_push.md).

## Contributing In General
Our project welcomes external contributions. If you have an itch, please feel
free to scratch it.

To contribute code or documentation, please submit a [pull request](https://github.com/foundation-model-stack/multi-nic-cni/pulls).

A good way to familiarize yourself with the codebase and contribution process is
to look for and tackle low-hanging fruit in the [issue tracker](https://github.com/foundation-model-stack/multi-nic-cni/issues).
Before embarking on a more ambitious contribution, please quickly [get in touch](#communication) with us.

**Note: We appreciate your effort, and want to avoid a situation where a contribution
requires extensive rework (by you or by us), sits in backlog for a long time, or
cannot be accepted at all!**

### Proposing new features

If you would like to implement a new feature, please [raise an issue](https://github.com/foundation-model-stack/multi-nic-cni/issues)
before sending a pull request so the feature can be discussed. This is to avoid
you wasting your valuable time working on a feature that the project developers
are not interested in accepting into the code base.

### Fixing bugs

If you would like to fix a bug, please [raise an issue](https://github.com/foundation-model-stack/multi-nic-cni/issues) before sending a
pull request so it can be tracked.

### Merge approval

The project maintainers use LGTM (Looks Good To Me) in comments on the code
review to indicate acceptance. A change requires LGTMs from two of the
maintainers of each component affected.

For a list of the maintainers, see the [MAINTAINERS.md](../Maintainers.md) page.

## Legal

Each source file must include a license header for the Apache
Software License 2.0. Using the SPDX format is the simplest approach.
e.g.

```
/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
```

We have tried to make it as easy as possible to make contributions. This
applies to how we handle the legal aspects of contribution. We use the
same approach - the [Developer's Certificate of Origin 1.1 (DCO)](https://github.com/hyperledger/fabric/blob/master/docs/source/DCO1.1.txt) - that the Linux® Kernel [community](https://elinux.org/Developer_Certificate_Of_Origin)
uses to manage code contributions.

We simply ask that when submitting a patch for review, the developer
must include a sign-off statement in the commit message.

Here is an example Signed-off-by line, which indicates that the
submitter accepts the DCO:

```
Signed-off-by: John Doe <john.doe@example.com>
```

You can include this automatically when you commit a change to your
local git repository using the following command:

```
git commit -s
```

## Setup
#### Clone the repo and enter the workspace

```bash
git clone https://github.com/foundation-model-stack/multi-nic-cni.git
cd multi-nic-cni-operator
```
#### Requirements
- [go](https://go.dev/doc/install)
- [operator-sdk](https://sdk.operatorframework.io/docs/installation/)
- utility tools
  - environment substitution *envsubst* ([gettext](https://www.gnu.org/software/gettext/))
  - YAML processor *yq* ([yq](https://mikefarah.gitbook.io/yq/))

## Testing
- Unit test
```bash
make test
```
- Local functional test in your local cluster
[Build and deploy locally](./local_build_push.md)
    * [Deploy network](./../user_guide/index.md#deploy-multinicnetwork-resource)
    * [Check all-to-all connections](./../user_guide/index.md#check-connections)
    * Test your new feature (if proposing new feature)

## Coding style guidelines
Follow [effective GO](https://go.dev/doc/effective_go).

Please run `make pr` before submitting PRs.
