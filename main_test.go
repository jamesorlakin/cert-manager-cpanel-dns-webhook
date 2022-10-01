package main

import (
	"testing"
	// "github.com/jetstack/cert-manager/test/acme/dns"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

// var (
// 	zone = "test-domain.com."
// )

func CreatesClientFromSecretValues(t *testing.T) {
	configJson := corev1.Secret{
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("password"),
			"apiToken": []byte("apiToken"),
		},
	}
	client, err := CreateClientFromSecretValues(&configJson, "zone", "cpanel")
	if err != nil {
		t.Error("Unexpected error")
	}

	assert.Equal(t, "username", client.Username)
	assert.Equal(t, "password", client.Password)
	assert.Equal(t, "apiToken", client.ApiToken)
	assert.Equal(t, "zone", client.DnsZone)
	assert.Equal(t, "cpanel", client.CpanelUrl)
}

func ReturnsErrorDueToMissingUsername(t *testing.T) {
	emptyConfig := corev1.Secret{
		Data: map[string][]byte{
			"password": []byte("password"),
		},
	}
	_, err := CreateClientFromSecretValues(&emptyConfig, "zone", "cpanel")
	assert.EqualError(t, err, "username field not present in secret")
}

func ReturnsErrorDueToMissingCredentials(t *testing.T) {
	emptyConfig := corev1.Secret{
		Data: map[string][]byte{
			"username": []byte("user"),
		},
	}
	_, err := CreateClientFromSecretValues(&emptyConfig, "zone", "cpanel")
	assert.EqualError(t, err, "username field not present in secret")
}

func TestRunsSuite(t *testing.T) {
	// TODO(jamesorlakin): Need to test main webhook logic. Not sure how to fake a k8s client secret?

	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.

	// fixture := dns.NewFixture(&customDNSProviderSolver{},
	// 	dns.SetResolvedZone(zone),
	// 	dns.SetAllowAmbientCredentials(false),
	// 	dns.SetManifestPath("testdata/my-custom-solver"),
	// )
	//need to uncomment and  RunConformance delete runBasic and runExtended once https://github.com/cert-manager/cert-manager/pull/4835 is merged
	//fixture.RunConformance(t)
	// fixture.RunBasic(t)
	// fixture.RunExtended(t)

}
