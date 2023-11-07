# Workflow for release

## Prerequisite Steps

* Update the `version` and `appVersion` fields in the `Chart.yaml` files located under the `charts/*` directories.

* Update the version in the `/VERSION` file.

* Ensure to set a version tag on the appropriate branch. The version should follow the below format:

  * v0.1.0-rc0 (Release Candidate 0)
  * v0.1.0-rc1 (Release Candidate 1)
  * v0.1.0 (Initial Release)
  * v0.2.0-rc0 (Second Version's Release Candidate 0)
  * v0.2.0 (Second Version's Initial Release)

## Tagging a Version

When a version tag, denoted as `vx.x.x`, is pushed, the subsequent automated process initiates:

1. The tag name is verified to match with `/VERSION`.
2. A branch named `release-vx.x.x` is created.
3. Docker images are built, tagged with the pushed tag, and then pushed to the GHCR (GitHub Container Registry).
4. A changelog is generated from historical PRs labeled with `pr/release/*` and added to the `changelogs` directory in the `github_pages` branch. This changelog PR should be labeled as `pr/release/robot_update_githubpage`. The changelog categorizes historical PR labels as follows:
    - PRs labeled `release/feature-new` are classified as "New Features".
    - PRs labeled `release/feature-changed` are classified as "Changed Features".
    - PRs labeled `release/bug` are classified as "Fixes".
5. A Helm chart package is built using the pushed tag and a PR is submitted to the `github_pages` branch. The chart can be retrieved using the command:
    ```
    helm repo add $REPO_NAME https://kdoctor-io.github.io/$REPO_NAME
    ```
6. Content from the `/docs` directory is submitted to the `/docs` directory of the `github_pages` branch.
7. A GitHub Release is created, which includes the Helm chart package and the changelog attached.
8. Finally, manual approval is required for the Helm chart PR and the changelog PR, both labeled as `pr/release/robot_update_githubpage`.
This process ensures version control, continuous integration, and delivery are maintained effectively.
