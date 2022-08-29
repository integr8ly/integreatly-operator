# Custom Domains
This will explain how to configure the operator to use a custom domain when the operator is not being installed via the addon.

In this guide the `rhmi.me` domain will be used as the example custom domain.

## Cluster Configuration
There are external steps required to configure a cluster to allow custom domains.

Required Items

* Domain Certificate
* A domain name
* CNAME edit access for a domain name


### Domain Certificate

Creating a self-signed certificate.
```shell
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
-keyout rhmi.me.key -out rhmi.me.crt \
-subj '/CN=apps.rhmi.me' -addext 'subjectAltName=DNS:*.apps.rhmi.me'
```
The `subjectAltName` is required by 3scale installation.
Values of the certificate can be examined as follows.
```shell
openssl x509 -noout -text -in rhmi.me.crt
```

The certificate is  added to a namespace on the cluster.
```shell
oc new-project certs
oc create secret tls rhmi-me-tls --cert=rhmi.me.crt --key=rhmi.me.key -n certs
```

On the cluster a Custom Domain CR needs to be created.
More information about the Custom Domain CR can be found in the [Openshift documentation](https://docs.openshift.com/dedicated/applications/deployments/osd-config-custom-domains-applications.html).
```shell
oc apply -f - <<EOF                                                             
---             
apiVersion: managed.openshift.io/v1alpha1
kind: CustomDomain
metadata:
  name: customdomain1
spec:
  domain: apps.rhmi.me 
  scope: External
  certificate:
    name: rhmi-me-tls
    namespace: certs
EOF
```

Once the Custom Domain CR is in a ready state there will be an endpoint value that is required by the CNAME record.

Checking the state of the Custom Domain CR.
```shell
oc get CustomDomain customdomain1 -o jsonpath='{.status.state}{"\n"}'
```

Getting the endpoint URL from the Custom Domain CR.
```shell
oc get CustomDomain customdomain1 -o jsonpath='{.status.endpoint}{"\n"}'
```

The endpoint URL will need to be added to a CNAME record for the domain in question.
Team leads should be able to help get access to the required resources.
This guide does not cover the steps required to configure the CNAME record in the DNS configuration.

## Configuring the operator installation.
Currently, the operator only allows the configuring of the custom domain during installation. 
Once the routes have being created within 3scale they can not be changed again.

Run the normal cluster prepare step, `make cluster/prepare/local`.
Patch the secret with the domain name.
```shell
echo -n "apps.rhmi.me" | base64 | xargs -I {} \
oc patch secret addon-managed-api-service-parameters \
-n redhat-rhoam-operator --type=json \
-p='[{"op": "replace", "path": /data/custom-domain_domain, "value": "{}"}]'
```
As this setup is using self-signed certificates the operator will need to be configured to allow this.
One way of doing this is to create the rhmi CR on cluster with the make target of `make deploy/integreatly-rhmi-cr.yml`.
If the rhmi CR has already been created it can be patched after the installation has started. 
Patch the CR as follows.

```shell    
oc patch rhmi rhoam -n redhat-rhoam-operator --type=json -p='[{"op": "replace", "path": /spec/selfSignedCerts, "value": true}]'
```

## Finished installation
After installation has complete the following information should be seen.

* The RHMI CR will have a status block for the custom domain settings. 
* The subdomain routing field in the RHMI CR spec will match the custom domain.
* Routes in the 3Scale namespace will have the custom domain set.
* RHSSO client will have the custom domain configured in the callbacks.
* Console link redirects to 3Scale custom domain URL. 