# Introduction

## copy

1. copy repo

   replace all 'egressgateway' to 'YourRepoName'

   replace all 'spidernet-io' and 'spidernet.io' to 'YourOrigin'

2. grep "====modify====" * -RHn --colour  and modify all of them

4. github seetings:

   spidernet.io  -> settings -> secrets -> actions -> grant secret to repo

   repo -> packages -> package settings -> Change package visibility

   create 'github_pages' branch, and repo -> settings -> pages -> add branch 'github_pages', directory 'docs'

   repo -> settings -> branch -> add protection rules for 'main' and 'github_pages'

5. redefine CRD in pkg/k8s/v1, and `make update_crd_sdk`, and code pkg/mybookManager

6. update api/v1/openapi.yaml and `update_openapi_sdk`

7. update charts/ , and images/ , and CODEOWNERS

8. enable third app

   codefactor: https://www.codefactor.io/dashboard

   sonarCloud: https://sonarcloud.io/projects/create

3. create badge for github/workflows/auto-nightly-ci.yaml, github/workflows/badge.yaml

## local develop

1. `make build_local_image`

2. `make e2e_init`

3. `make e2e_run`

4. check proscope, browser vists http://NodeIP:4040

5. apply cr

        cat <<EOF > mybook.yaml
        apiVersion: egressgateway.spidernet.io/v1
        kind: Mybook
        metadata:
          name: test
        spec:
          ipVersion: 4
          subnet: "1.0.0.0/8"
        EOF
        kubectl apply -f mybook.yaml

## chart develop

helm repo add rock https://spidernet-io.github.io/egressgateway/

