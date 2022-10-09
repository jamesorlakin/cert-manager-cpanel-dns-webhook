package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jamesorlakin/cert-manager-cpanel-dns-webhook/cpanel"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
	log "github.com/sirupsen/logrus"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	log.SetLevel(log.DebugLevel)
	log.Info("cert-manager CPanel webhook solver starting, v0.2.0")
	if GroupName == "" {
		log.Panic("GROUP_NAME must be specified as an environment variable")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&customDNSProviderSolver{},
	)
}

// customDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver`
// interface.
type customDNSProviderSolver struct {
	client *kubernetes.Clientset

	// CPanel requires the zone serial in requests. This value could be sent in two requests concurrently
	// but only one will win and actually be persisted in the zone file - there's not even an error back from CPanel.
	// We therefore use this mutex to disallow concurrent requests to CPanel if multiple DNS names are given.
	mutex sync.Mutex
}

// customDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type customDNSProviderConfig struct {
	// These fields will be set by users in the
	// `issuer.spec.acme.dns01.providers.webhook.config` field.

	// The URL to a CPanel instance without a trailing slash, e.g. https://cpanel.mydomain.com
	CpanelUrl string `json:"cpanelUrl"`

	// A reference to a secret, in the form "namespace/secret-name"
	// This secret should have data of 'username' and 'password'
	SecretRef string `json:"secretRef"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *customDNSProviderSolver) Name() string {
	return "cpanel-solver"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *customDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	log.Infof("Got request to present: %+v", ch)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	log.Debugf("Presenting %+v", ch)
	cpanel, err := c.getDnsClient(ch)
	if err != nil {
		log.Error("Could not get cpanelClient")
		return err
	}

	err = cpanel.SetDnsTxt(ch.ResolvedFQDN, ch.Key)
	log.Debugf("Present complete %+v", ch)
	return err
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *customDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	log.Infof("Got request to clean up: %+v", ch)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	log.Debugf("Deleting %+v", ch)
	cpanel, err := c.getDnsClient(ch)
	if err != nil {
		log.Error("Could not get cpanelClient")
		return err
	}

	err = cpanel.ClearDnsTxt(ch.ResolvedFQDN, ch.Key)
	log.Debugf("CleanUp complete %+v", ch)
	return err
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *customDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		log.Error("couldn't get Kubernetes clientset", err)
		return err
	}

	c.client = cl
	return nil
}

// Lookup the secret in the config and get values out of it to construct a client instance
func (c *customDNSProviderSolver) getDnsClient(ch *v1alpha1.ChallengeRequest) (*cpanel.CpanelClient, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	log.Infof("Decoded webhook configuration %+v", cfg)
	if cfg.CpanelUrl == "" {
		return nil, errors.New("dnsZone or cpanelUrl wasn't provided")
	}
	secretRefSplit := strings.Split(cfg.SecretRef, "/")
	if len(secretRefSplit) != 2 {
		return nil, errors.New("expected secretRef to be in the form namespace/name")
	}
	secretNamespace := secretRefSplit[0]
	secretName := secretRefSplit[1]

	log.Debugf("Fetching contents of secret %s from namespace %s", secretName, secretNamespace)
	secret, err := c.client.CoreV1().Secrets(secretNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Error("could not get secret", err)
		return nil, err
	}

	client, err := CreateClientFromSecretValues(secret, ch.ResolvedZone, cfg.CpanelUrl)
	return client, err
}

func CreateClientFromSecretValues(secret *corev1.Secret, dnsZone, cpanelUrl string) (*cpanel.CpanelClient, error) {
	usernameBytes, ok := secret.Data["username"]
	if !ok {
		err := errors.New("username field not present in secret")
		log.Error(err)
		return nil, err
	}
	passwordBytes, passwordOk := secret.Data["password"]
	apiTokenBytes, apiOk := secret.Data["apiToken"]
	if !passwordOk && !apiOk {
		err := errors.New("password or API token field not present in secret")
		log.Error(err)
		return nil, err
	}

	username := string(usernameBytes)
	password := string(passwordBytes)
	apiToken := string(apiTokenBytes)

	log.Info("Got credentials from secret")

	cpanel := &cpanel.CpanelClient{
		DnsZone:   dnsZone,
		CpanelUrl: cpanelUrl,
		Username:  username,
		Password:  password,
		ApiToken:  apiToken,
	}
	return cpanel, nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (customDNSProviderConfig, error) {
	cfg := customDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %+v", err)
	}

	return cfg, nil
}
