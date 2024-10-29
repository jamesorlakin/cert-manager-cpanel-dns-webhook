# cert-manager CPanel DNS Webhook

A simple webhook DNS solver for cert-manager using the [CPanel UAPI](https://api.docs.cpanel.net/openapi/cpanel/operation/dns-mass_edit_zone/) for those of us stuck using their webhost's CPanel for DNS.

This is based off the [example](https://github.com/cert-manager/webhook-example) webhook.

## Getting started

1. Install `cert-manager`. This was developed when running v1.8 but it should work reasonably across versions. This is assumed to be in the `cert-manager` namespace, if not you'll need to tweak the Helm values.
2. Install this webhook. There's a Helm chart in `deploy/chart` (`helm install cpanel-webhook .`), or `kubectl apply -f https://raw.githubusercontent.com/jamesorlakin/cert-manager-cpanel-dns-webhook/master/deploy/v0.3.0.yaml` will install this in the `cert-manager` namespace.
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
      # Or, instead of a password in v0.2.0+, create and use an API token from CPanel's Security section:
      apiToken: ABCDEF1234567890ABCDEFABCDEF1234567890
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
              groupName: jameslakin.co.uk # Must match the group name in the Helm chart (this is the default and shouldn't need changing to your own domain)
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
      name: example-cpanel-cert
    spec:
      secretName: example-cpanel-cert
      issuerRef:
        name: letsencrypt-staging
        kind: ClusterIssuer
      dnsNames:
      - '*.whatever.my-super-website.com'
    ```
