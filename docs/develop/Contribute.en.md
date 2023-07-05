# How to contribute

First of all, thank you for your interest in contributing to our project! We appreciate all the help and support from the community. This document outlines the process for contributing to our Kubernetes operator project. Please read through the guidelines carefully before submitting any changes.

## Code of conduct

To ensure a positive and supportive environment for all contributors, please adhere to our Code of Conduct. By participating in this project, you agree to abide by its terms.

## Getting started

Fork the Repository: To contribute, start by forking the repository to your own GitHub account. This will create a copy of the project that you can modify as needed.

Clone the Fork: After forking the repository, clone your fork to your local machine using Git. This will allow you to work on the project locally.

```shell
git clone https://github.com/your-username/your-project-name.git
```

Create a Branch: Before starting to work on your changes, create a new branch for each feature or bugfix. This helps to keep your changes separated and organized. Use a descriptive name for your branch.

```shell
git checkout -b your-new-branch-name
```
## Making changes

Update Your Fork: Before making any changes, make sure your fork is up to date with the upstream repository.

```shell
git remote add upstream https://github.com/original-owner/your-project-name.git
git fetch upstream
git merge upstream/main
```

Test Your Changes: After making changes, ensure that they do not break the project by running tests. Be sure to test your changes in a Kubernetes environment if possible.

Commit Your Changes: When you're satisfied with your changes, commit them using a clear and concise commit message. This helps the maintainers understand the purpose of your changes.

```shell
git add .
git commit -m "Your commit message"
```

Push Your Changes: After committing your changes, push them to your fork on GitHub.

```shell
git push origin your-new-branch-name
```

## Submitting a pull request

Create a Pull Request: Once your changes are pushed, create a pull request from your fork to the upstream repository. Make sure the base branch is set to main and the head branch is your feature or bugfix branch.

Describe Your Changes: In the pull request description, provide a clear and concise description of the changes you made. Include any relevant issue numbers and explain how your changes address the issue or add a new feature.

Wait for a Review: After submitting your pull request, wait for a project maintainer to review your changes. They may request changes or provide feedback on your work. Be sure to address any feedback and update your pull request as needed.

Merge: Once your changes have been reviewed and approved, a project maintainer will merge your pull request. Congratulations, you've successfully contributed to the project!