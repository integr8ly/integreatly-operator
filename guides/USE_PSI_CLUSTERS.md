# Using OCP4 Clusters on PSI

You can use OCP4 clusters that are created on PSI(PnT Shared Infrastructure) for development work. There are a few benefits:

* It is completely free (PSI is hosted and maintained by Red Hat)
* No expire date for the clusters, so you can have long running ones if you need
* You can have multiple OCP4 clusters created if you need.

But there are also a few limitations:

* Red Hat VPN is required to create and access these clusters
* Only self-signed certificates are supported by these clusters

# Requesting a cluster

You can request a PSI OCP4 cluster using the [Flexy Jenkins Job](https://mastern-jenkins-csb-openshift-qe.cloud.paas.psi.redhat.com/job/Launch%20Environment%20Flexy). 

In order to trigger the job, you need to request build permission to the instance first. 

## Request build permission

You need to (The instruction is also available on the Jenkins job page):

1. Go to this [Service Now Page](https://redhat.service-now.com/rh_ess/catalog.do?v=1&uri=com.glideapp.servicecatalog_cat_item_view.do%3Fv%3D1%26sysparm_id%3Defc26a27053142004c7104229f8248df%26sysparm_link_parent%3D34feb8be2b50c9004c71dc0e59da1553%26sysparm_catalog%3De0d08b13c3330100c8b837659bba8fb4&sysparm_document_key=sc_cat_item,efc26a27053142004c7104229f8248df) to create a SNOW ticket.
   
    1. Add your name to the `Requested for` field
    2. In the `LDAP Group Details` section, enter `aos-qe-installer` to the `Available` field, select it and click on the `>` button to add it
    3. For the `Provide details about why you need to be added to these groups`, enter `Create OCP4 clusters on PSI`. 

2. Submit the request and wait. It normally takes about 24-48 hours for the ticket to be resolved.

Once the SNOW ticket is resolved, you will be able to trigger the Jenkins job. 

## Create a PSI OCP4 cluster

To create a PSI OCP4 cluster using the [Flexy Job](https://mastern-jenkins-csb-openshift-qe.cloud.paas.psi.redhat.com/job/Launch%20Environment%20Flexy):

1. Login the Jenkins instance using your Kerberos username and password
2. Click on `Build with parameters` on the left side, and use the following parameters:
   1. `INSTANCE_NAME_PREFIX`: use something that is unique to you. It should be less than 14 characters.
   2. `VARIABLES_LOCATION`: enter `private-openshift-misc/v3-launch-templates/functionality-testing/aos-4_4/ipi-on-osp/versioned-installer`. This will install OpenShift version 4.4. Modify `aos-4_4` to values that match an OpenShift version if you want to use a different one. E.g. `aos-4_3`. Use the link in the field instruction to see available templates. 
   3. Set `LAUNCHER_VARS`, `BUSHSLICER_CONFIG` fields to empty if they are not.
   4. You should use the default values for `BUSHSLICER_LOG_LEVEL`, `REPO_OWNER`, `BRANCH`, `OPENSHIFT_MISC_BRANCH`, `OPENSHIFT_ANSIBLE_URL`, `OPENSHIFT_ANSIBLE_BRANCH`, `AOS_ANSIBLE_URL` and `AOS_ANSIBLE_BRANCH` fields.
   5. Update `JENKINS_SLAVE_LABEL` to use the right slave based on the instructions. For example, if you are requesting an OpenShift 4.4 cluster, change the value to `oc44`.
3. Click on `Build` to start the job. In about 30-40 minutes, the cluster will be ready.
4. Once the job is completed successfully, you will be able to find all the files that you need to access the cluster (like `kubeconfig`) in the `Build Artifacts` of the job. You will be able to find the URL of the cluster and kubeadmin credentials in the `.openshift_install.log` file (towards the end of the file).

Once the cluster is ready, you can then install RHMI operator to it by following the instructions in the [README file](../README.md).

## Destroy the cluster

It is recommended that the cluster is destroyed once it's not needed anymore to release resources. To destroy the cluster, use [this Jenkins job](https://mastern-jenkins-csb-openshift-qe.cloud.paas.psi.redhat.com/job/Remove%20VMs/):

1. Find the id of the build that created the cluster. You can find the id in the url when you view the details of a build, something like `91643`. Copy it.
2. Trigger a new build of the [Remove VMs job](https://mastern-jenkins-csb-openshift-qe.cloud.paas.psi.redhat.com/job/Remove%20VMs), and enter the build id as the parameter. 