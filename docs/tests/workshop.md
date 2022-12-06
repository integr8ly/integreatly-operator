# Workshop postgres/redis instances
The workshop type is being used for creating on cluster, throwaway postgres/redis instances for troubleshooting. 
While this isn't functionality used in production, there are some manual test cases and sops that rely on it.

## Usage
To create a workshop instance, log in to a cluster and issue one of the following commands:

### Postgres
```shell
cat << EOF | oc create -f - -n redhat-rhoam-operator
  apiVersion: integreatly.org/v1alpha1
  kind: Postgres
  metadata:
    name: throw-away-postgres
    labels:
      productName: productName
  spec:
    secretRef:
      name: throw-away-postgres-sec
    tier: development
    type: workshop
EOF
```

### Redis
```shell
cat << EOF | oc create -f - -n redhat-rhoam-operator
  apiVersion: integreatly.org/v1alpha1
  kind: Redis
  metadata:
    name: throw-away-redis
    labels:
      productName: productName
  spec:
    secretRef:
      name: throw-away-redis-sec
    tier: development
    type: workshop
EOF
```