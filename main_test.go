package main

import (
	"testing"
	// "github.com/jetstack/cert-manager/test/acme/dns"
)

var (
	zone = "test-domain.com."
)

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
