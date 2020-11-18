---
components:
  - product-3scale
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.7.0
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
estimate: 2h
tags:
  - destructive
---

# K02 - Run performance test against 3Scale

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

1. Promote `API` API in 3scale to production environment
   1. Login to 3scale console as dedicated admin
   2. Open `Integration` for `API`
   3. Make note of the service ID in URL (e.g. `2`)
   4. Promote `API` to Production environment
   5. Make sure you can `curl` it (optional)
   6. Open Settings by clicking cog icon in top right corner
   7. Copy API key
2. Prepare VM in OpenStack or AWS
   - for OpenStack you can use integreatly-qe jenkins slave (there should be few already running), if you do not have access contact integreatly-qe for VM IPs:
     1. Login to [OpenStack](https://rhos-d.infra.prod.upshift.rdu2.redhat.com/)
     2. In the top right click `Projects` dropdown and choose `integreatly-qe` project
     3. Open `Instances` tab
     4. Choose one of `upshift-rhel-...` instances and copy its IP address
     5. Clone [integreatly-qe repo](https://gitlab.cee.redhat.com/integreatly-qe/integreatly-qe)
     6. `ssh -i /path/to/integreatly-qe/infra/integr8ly jenkins@<IP_ADDRESS>`
     7. In the VM run:
        1. `sudo su`
        2. `setenforce 0`
        3. `exit`
        4. `exit`
   - for AWS some info is in [this JIRA](https://issues.redhat.com/browse/INTLY-5037?focusedCommentId=13961287&page=com.atlassian.jira.plugin.system.issuetabpanels%3Acomment-tabpanel#comment-13961287)
3. Deploy injector
   1. Clone [3scale/perftest-toolkit](https://github.com/3scale/perftest-toolkit) to your local machine
   2. `cd perftest-toolkit/deployment`
   3. Edit `hosts` file - `injector` should point to your VM
      > `injector` line should look something like: `injector ansible_host=10.0.154.25 ansible_user=jenkins ansible_ssh_private_key_file=/path/to/integreatly-qe/infra/integr8ly`
   4. Edit `roles/user-traffic-reader/defaults/main.yml` file:
      - `threescale_portal_endpoint` should be similar to: `https://<3SCALE_API_KEY>@3scale-admin.apps.rhmi-byoc.f7a1.s1.devshift.org/`
      - `threescale_services` should be ID of your `API` service (noted in step 1.3.)
   5. Deploy injector: `ansible-playbook -i hosts injector.yml`. If you recieve timout errors like `Timeout (12s) waiting for privilege escalation prompt:` at this step, adjust the timeout value in your anisble config to a higher value e.g `30 seconds`. The config is usually at `/etc/ansible/ansible.cfg and rerun this step.
4. Run the tests
   1. Edit `run.yml`:
      - change `RPS` to 2000
      - change `DURATION` to 600
      - change `THREADS` to 50
   2. Run the tests: `ansible-playbook -i hosts run.yml`
5. Compare results
   1. Copy results: `scp -r -i /path/to/integreatly-qe/infra/integr8ly jenkins@<VM_IP_ADDRESS>:/opt/3scale-perftest/reports ./reports`
   2. Open `./reports/report/index.html`
   3. Compare results with previous results - stored in this repo - `fixtures/3scale-perf-results.tar.gz`
      > Results should be similar
