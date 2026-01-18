<!--
Thank you for helping to improve Crossplane!

Please read through https://git.io/fj2m9 if this is your first time opening a
Crossplane pull request. Find us in https://slack.crossplane.io/messages/dev if
you need any help contributing.
-->

### Description of your changes

<!--
Briefly describe what this pull request does. Be sure to direct your reviewers'
attention to anything that needs special consideration.

We love pull requests that resolve an open Crossplane issue. If yours does, you
can uncomment the below line to indicate which issue your PR fixes, for example
"Fixes #500":

-->
Fixes #

I have:

- [ ] Read and followed Crossplane's [contribution process].
- [ ] Followed the git conventional commit message format.
- [ ] Made sure all changes are covered by proper tests, reaching a coverage of at least 80% when applicable.
- [ ] Run `make reviewable` to ensure this PR is ready for review.
- [ ] Added `backport release-x.y` labels to auto-backport this PR if necessary.

### How has this code been tested

I have:

- [ ] Successfully built and ran the provider locally against a kubernetes cluster.
- [ ] Successfully created, updated, and deleted resources of the types I changed / created.
- [ ] Ensured reconciliation loops for the changed / created resource complete without error.
  - [ ] Creation
  - [ ] Update
  - [ ] Deletion
- [ ] Deleted the resource on the app side to ensure the provider correctly handles
  unexpected drift. (should result in recreation of the resource if applicable)
- [ ] Updated the resource on the app side to ensure the provider correctly handles
  unexpected drift. (should result in an update of the resource if applicable)

<!--
Before reviewers can be confident in the correctness of this pull request, it
needs to tested and shown to be correct. Briefly describe the testing that has
already been done or which is planned for this change.
-->

[contribution process]: https://git.io/fj2m9
