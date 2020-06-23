# Release Process

Our release process has been automated and the Jenkins pipeline can be found [here](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Delorean/job/rhmi-release/). It will perform the following steps:

1. Create a new branch off the base branch
2. Execute the [pre-release script](./scripts/prepare-release.sh)
3. Commit changes and create a PR
4. Wait for the PR to be reviewed.
5. Merge the PR and then create the release tag for the base branch HEAD, and wait for the image to be available on quay.io and tag it.
6. Create the PR against the managed-tenants repo. Stage channel first, then later on the same changes should be made to the edge and stable channels.

NOTE: Red Hat VPN is required to access.

# Prepare for Patch Releases

In order to create patch releases, we need to make sure a release branch is created first and it is configured properly.
   
Run the following pipeline to create a release branch: [rhmi-prepare-patch-release](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Delorean/job/rhmi-prepare-patch-release/)

- Fill in the `version` with the patch version that you are going to create (example: `2.1.1`, `2.3.1`)

The above pipeline will create the release branch if it doesn't exist, and update or create the build configuration for the release branch in the [openshift/release](https://github.com/openshift/release) repository.

Once the release branch is created, you can make changes to it. The same changes should be cherry-picked to master branch as well.

Then follow steps described in the next section to create a release when it's ready.

# All Releases

To perform a release, you should:
1. Login to the [Jenkins instance](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Delorean/job/rhmi-release/) and trigger the job by clicking on **Build with Parameters**.
2. For the job parameters:
   1. Fill in the `version` value. It should follow the semver spec, something like `2.1.0`, `2.1.0-er1`, `2.1.1-rc1` etc.
   2. Change the value of `integreatlyOperatorBaseBranch` to the release branch if it's a patch release.
   3. Check the `updateManagedTenantsOnly` option if you only need to update the `managed-tenants` repo to push a release to edge or stable channels 
   4. Leave the rest as they are
3. Click on the *Build* button to start the release. Switch to the blue ocean view to see the progress of the build.
4. At some point, you will be prompted to review the PR. You should then open the PR link, review it (or make any additional changes if required), but you don't need to merge it.
5. Once done, you can click on the "Approve" link to continue the pipeline. If it doesn't work for you in Chrome, try Firefox or use the Blue Ocean view. 
6. Wait for the build to finish, and at the end you should see a merge request link to the managed-tenant repo.
7. Review the MR (and make any additional changes if required). Ping reviewers on the PR once it's ready. 
8. When the final release is done, make sure close the merge blocker issue on Github to allow Prow to start merging PRs back to the release branch.
9. If this is a patch release and a new ClusterServiceVersion (CSV) is generated, please make sure adding the new CSV files back to the master branch. This needs to be done manually for now and we will add automation in the future. The following is an example how you can do it:
    
    ```
    git checkout upstream/release-v2.1 -- deploy/olm-catalog/integreatly-operator/
    ```

If `updateManagedTenantsOnly` is checked, you only need to follow step 6-7.
