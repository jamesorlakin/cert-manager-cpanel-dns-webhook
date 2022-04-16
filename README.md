# cert-manager CPanel DNS Webhook

A webhook DNS solver for cert-manager using the [CPanel UAPI](https://api.docs.cpanel.net/openapi/cpanel/operation/dns-mass_edit_zone/) for those of us stuck using their webhost's CPanel for DNS.

## Getting started

1. Install `cert-manager`. This was developed when running v1.8 but it should work reasonably across versions. This is assumed to be in the `cert-manager` namespace, if not you'll need to tweak the Helm values.
2. Install this webhook. There's a Helm chart in `deploy/chart` (`helm install cpanel-webhook .`), or `kubectl apply https://raw.githubusercontent.com/jamesorlakin/cert-manager-cpanel-dns-webhook/master/deploy/v0.1.0.yaml` for the `cert-manager` namespace.
3. Create a secret containing your CPanel credentials.
    ```yaml
    apiVersion: v1
    kind: Secret
    type: Opaque
    metadata:
      name: some-cpanel-credentials
      namespace: cert-manager
    stringData:
      username: my-cpanel-user
      password: my-cpanel-password
    ```
4. Create an ACME issuer referencing the webhook, e.g.:
    ```yaml
    apiVersion: cert-manager.io/v1
    kind: ClusterIssuer
    metadata:
      name: letsencrypt-staging
    spec:
      acme:
        server: https://acme-staging-v02.api.letsencrypt.org/directory
        email: my-acme-email@yourself.com
        privateKeySecretRef:
          name: letsencrypt-staging
        solvers:
        - dns01:
            # The fun bit:
            webhook:
              groupName: jameslakin.co.uk # Must match the group name in the Helm chart (this is the default)
              solverName: cpanel-solver # Don't change
              config:
                cpanelUrl: https://cpanel.my-super-website.com # No trailing slash
                secretRef: cert-manager/some-cpanel-credentials # In the form namespace/secret-name
    ```
5. ...issue certificates:
    ```yaml
    apiVersion: cert-manager.io/v1
    kind: Certificate
    metadata:
      name: example-cpanel-cert-5
    spec:
      secretName: example-cpanel-cert-5
      issuerRef:
        name: letsencrypt-staging
        kind: ClusterIssuer
      dnsNames:
      - '*.whatever.my-super-website.com'
    ```
