# Release Workflow

## Pre-steps

* Update `version` and `appVersion` fields in `charts/*/Chart.yaml`
* Update version in `/VERSION`
* Set a version tag on the correct branch. The version should follow the pattern:
    * v0.1.0-rc0
    * v0.1.0-rc1
    * v0.1.0
    * v0.1.1
    * v0.1.2
    * v0.2.0-rc0
    * v0.2.0

## Push a Version Tag

When a tag vx.x.x is pushed, the following steps will automatically run:

1. Verify that the tag name matches the `/VERSION`
2. Create a branch named `release-vx.x.x`
3. Build the images with the pushed tag and push them to the ghcr registry
4. Generate the changelog based on historical PRs labeled as `release/*`
    - Submit the changelog file to the `changelogs` directory of the `github_pages` branch, with PR labeled as `pr/release/robot_update_githubpage`
    - Changelogs are generated based on historical PR labels:
        - Label `release/feature` will be classified as "Changed Features"
        - Label `release/bug` will be classified as "Fixes"
5. Build the chart package with the pushed tag and submit a PR to the `github_pages` branch
    - Retrieve the chart with the command `helm repo add $REPO_NAME https://spidernet-io.github.io/$REPO_NAME`
6. Submit `/docs` to the `/docs` directory of the `github_pages` branch
7. Create a GitHub Release with the chart package and changelog attached
8. Manually approve the chart PR labeled as `pr/release/robot_update_githubpage` and the changelog PR labeled as `pr/release/robot_update_githubpage`
