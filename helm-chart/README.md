# Training Application Helm Chart

## Prerequisites for the value `persistMetaInfo`

- A default storage class has to exist in the cluster.

## Prerequisites for the value `ingress.enabled`

- The IngressClass with the name `nginx` has to exist in the cluster
- CertManager has to exist in the cluster
- The ClusterIssuer with the name `letsencrypt-issuer` has to exist in the cluster

## Prerequisites for the value `gateway.enabled`

- The Gateway API CRDs have to exist in the cluster
- A Gateway has to exist in the cluster.
- On doing the Helm release the HTTPRoute hast to be configured properly via eg `values.yaml` to make yse of the Gateway.
