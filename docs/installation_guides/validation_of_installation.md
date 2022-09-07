# Validate installation 

Use following commands to validate that installation succeeded:

For `RHMI` (managed): `oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq .status.stage`

For `RHOAM` (managed-api): `oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage `

For `RHOAM Multitenant` (multitenant-managed-api): `oc get rhmi rhoam -n sandbox-rhoam-operator -o json | jq .status.stage `

Once the installation completed the command wil result in following output:  
```yaml
"complete"
```