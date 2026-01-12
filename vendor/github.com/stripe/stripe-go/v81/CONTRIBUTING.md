
# Contributing

We welcome bug reports, feature requests, and code contributions in a pull request.

For most pull requests, we request that you identify or create an associated issue that has the necessary context. We use these issues to reach agreement on an approach and save the PR author from having to redo work. Fixing typos or documentation issues likely do not need an issue; for any issue that introduces substantial code changes, changes the public interface, or if you aren't sure, please find or [create an issue](https://www.github.com/stripe/stripe-go/issues/new/choose).

## Contributor License Agreement

All contributors must sign the Contributor License Agreement (CLA) before we can accept their contribution. If you have not yet signed the agreement, you will be given an option to do so when you open a pull request. You can then sign by clicking on the badge in the comment from @CLAassistant.

## Generated code

This project has a combination of manually maintained code and code generated from our private code generator. If your contribution involves changes to generated code, please call this out in the issue or pull request as we will likely need to make a change to our code generator before accepting the contribution.

To identify files with purely generated code, look for the comment `File generated from our OpenAPI spec.` at the start of the file. Generated blocks of code within hand-written files will be between comments that say `The beginning of the section generated from our OpenAPI spec` and `The end of the section generated from our OpenAPI spec`.

## Compatibility with supported language and runtime versions

This project supports [many different langauge and runtime versions](README.md#requirements) and we are unable to accept any contribution that does not work on _all_ supported versions. If, after discussing the approach in the associated issue, your change must use an API / feature that isn't available in all supported versions, please call this out explicitly in the issue or pull request so we can help figure out the best way forward.

## Set up your dev environment

Please refer to this project's [README.md](README.md#development) for instructions on how to set up your development environment.

