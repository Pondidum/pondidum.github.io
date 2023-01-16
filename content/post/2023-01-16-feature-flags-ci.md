---
title: Feature Flags in a CI Pipeline
tags: [ "cli", "feature flags", "ci" ]
---

Feature flags are a great tool for helping software development; they provide controlled feature rollouts, facilitate A/B testing, and help decouple [deployment from release][deploy-not-release].  So when it comes to building our software, why do we treat the CI pipeline without the same level of engineering as the production code?

So, why not use feature flags in your CI pipeline?

## TLDR

Reduce the risk of breaking a CI pipeline for all of a project's developers by using the [flagon] CLI to query Feature Flags, opting developers into and out of new CI features and processes by targeting groups of developers or branch naming patterns.

## What would we use them for?

There are a few things that spring to mind that we could use feature flags for:

- Migrating CI system
- Job migration
- Replacing a step
- Trying a new step

## Why would using flags for this help?

The answer is risk reduction.  I don't want to break parts of the build and deployment process for everyone in the project when I make a mistake in the pipelines, and a way to help mitigate that risk is feature flags.

With a feature flag, I can quickly opt people into or out of changes to the CI system, meaning that if something goes wrong, the impact is minimal.  It also allows me to monitor the effects of new vs old by having the flag states stored in our OTEL traces.  This lets me ask and answer questions like: is it faster?  Is it more reliable?  Does it work?

One of the most significant risks is migrating from one CI system to another, which is exactly what I have been doing recently.  We are leaving `Truly Awful CI` and migrating to `Github Actions`.  Let's see how that goes.

## Migrating From Old to New CI

The CI process, on a high level, looks like this.  The three types of deployment are `ephemeral`, which are short-lived environments named after the branch which created them, `development`, which is the common development environment, and `production`, which is the live application.  The `production` and `development` environments are deployed to whenever something is merged to `main`, and `ephemeral` is for any other branch.

{{<mermaid align="left">}}
graph LR

    clone --> build --> test --> publish-container

    publish-container --> |$BRANCH != 'main'| trigger-deploy-ephemeral
    publish-container --> |$BRANCH == 'main'| trigger-deploy-development
    publish-container --> |$BRANCH == 'main'| trigger-deploy-production
{{</mermaid>}}

To phase the changeover to GitHub Actions, I am using the [flagon] CLI to access our feature flags.  The query uses both the user id (committer email) and branch name so that I can target rollouts based on a group of users or perhaps with a branch name pattern.

First, I create a duplicate workflow in GitHub Actions.  The docker container is published to a different tag, and all deployments have a feature flag condition added to them:

```yaml
jobs:
  flags:
    outputs:
      enable_ephemeral: ${{ steps.query.outputs.enable_ephemeral }}
    steps:
    - name: Query Flags
      id: query
      run: |
        ephemeral=$(flagon state "ci-enable-gha-deployment" "false" \
          --user "${email}" \
          --attr "branch=${branch}" \
          --output "template={{.Value}}" || true)

        echo "enable_ephemeral=${ephemeral}" >> "${GITHUB_OUTPUT}"

  # build, test, etc.

  deploy_ephemeral:
    uses: ./.github/workflows/deploy.yaml
    needs:
    - flags
    - build
    if: ${{ github.ref_name != 'master' && needs.flags.outputs.enable_ephemeral == 'true' }}
    with:
      target_env: ephemeral
```

Then update the old CI pipeline to wrap the deployment trigger with a flag query:

```bash
if ! flagon "ci-enable-gha-deploy" "true" \
  --user "${email}" \
  --attr "environment=ephemeral" \
  --attr "BRANCH=${branch}"; then

  awful-ci trigger "deploy - ephemeral" \
    --sha "${commit}" \
    --branch "${branch}"
fi
```

Note that the two CI systems are both querying the same flag, but the old system defaults to active, and the new system defaults to inactive.  This means that if the flagging service (LaunchDarkly in this case) cannot be reached, only one of the systems will be doing the deployment.

## Rolling out

The plan for starting the switchover was as follows:

1.  `ephemeral` for just me
2.  `ephemeral` environment for a small group of developers
4.  `ephemeral` for everyone
5.  `development` for everyone
6.  `production` for everyone
7.  WAIT
8.  Remove old implementation, remove flags

During the rollout, the flag was switched on and off for various stages as small bugs were found.

For example, I discovered that the deployments only looked like they were working in GitHub Actions due to some artefacts still being uploaded to CDN by the old CI system.

## Take Away

Based on my experience of using flags in this migration, it is a technique that I will be using more in the future when updating our CI pipelines.

[deploy-not-release]: /2022/11/02/deploy-doesnt-mean-release/
[flagon]: https://github.com/pondidum/flagon
