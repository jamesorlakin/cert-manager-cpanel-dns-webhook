package cpanel

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type DummyHttp struct {
	requests       []*http.Request
	responseBodies []string
}

// Implement RoundTrip so that it can be substituted into a HTTP Client
func (f *DummyHttp) RoundTrip(req *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, req)

	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(f.responseBodies[len(f.requests)-1])),
		Header:     make(http.Header),
	}, nil
}

const expectedUsernamePasswordAuthorization = "Basic dXNlcjpwYXNzd29yZA=="
		DnsZone:   "test-domain.com.",
		Username:  "user",
		Password:  "password",
		CpanelUrl: "https://cpanel.test-domain.com",
		httpClient: http.Client{
			Transport: mock,
		},
	}
}

func TestPresentCreate(t *testing.T) {
	mockClient := DummyHttp{
		responseBodies: []string{
			// SOA serial of 2022040507:
			`{
				"data": [
					{
						"line_index": 3,
						"type": "record",
						"data_b64": [
							"bnMxLnN0YWJsZWhvc3QuY29tLg==",
							"YWxlcnRzLnN0YWJsZWhvc3QuY29tLg==",
							"MjAyMjA0MDUwNQ==",
							"ODY0MDA=",
							"NzIwMA==",
							"MzYwMDAwMA==",
							"MTgwMA=="
						],
						"dname_b64": "amFtZXNsYWtpbi5jby51ay4=",
						"record_type": "SOA",
						"ttl": 86400
					}
				]
			}`,
			`{}`, // Errors needs to be empty, that's all
		},
	}
	client := NewClientWithMock(&mockClient)

	err := client.SetDnsTxt("dummy.test-domain.com.", "test-value")
	assert.NoError(t, err)

	// Expect 2 requests (one for zone info, one to create)
	assert.Len(t, mockClient.requests, 2)

	// Create
	request := mockClient.requests[1] // Second request
	expectedCreateUrl := `https://cpanel.test-domain.com/execute/DNS/mass_edit_zone?zone=test-domain.com&serial=2022040505&add=%7B%22dname%22%3A%22dummy%22%2C%22ttl%22%3A300%2C%22record_type%22%3A%22TXT%22%2C%22data%22%3A%5B%22test-value%22%5D%7D`
	assert.Equal(t, expectedCreateUrl, request.URL.String())
	assert.Equal(t, expectedUsernamePasswordAuthorization, request.Header["Authorization"][0])
}

func TestPresentNoCreate(t *testing.T) {
	mockClient := DummyHttp{
		responseBodies: []string{
			// SOA serial of 2022040507.
			// TXT record of dummy/test-value
			`{
				"data": [
					{
						"line_index": 3,
						"type": "record",
						"data_b64": [
							"bnMxLnN0YWJsZWhvc3QuY29tLg==",
							"YWxlcnRzLnN0YWJsZWhvc3QuY29tLg==",
							"MjAyMjA0MDUwNQ==",
							"ODY0MDA=",
							"NzIwMA==",
							"MzYwMDAwMA==",
							"MTgwMA=="
						],
						"dname_b64": "amFtZXNsYWtpbi5jby51ay4=",
						"record_type": "SOA",
						"ttl": 86400
					},
					{
						"line_index": 18,
						"type": "record",
						"data_b64": [
							"dGVzdC12YWx1ZQ=="
						],
						"dname_b64": "ZHVtbXk=",
						"record_type": "TXT",
						"ttl": 1
					}
				]
			}`,
		},
	}
	client := NewClientWithMock(&mockClient)

	err := client.SetDnsTxt("dummy.test-domain.com.", "test-value")
	assert.NoError(t, err)

	// Expect 1 request (only zone info, no create)
	assert.Len(t, mockClient.requests, 1)

	request := mockClient.requests[0]
	expectedCreateUrl := `https://cpanel.test-domain.com/execute/DNS/parse_zone?zone=test-domain.com`
	assert.Equal(t, expectedCreateUrl, request.URL.String())
	assert.Equal(t, expectedUsernamePasswordAuthorization, request.Header["Authorization"][0])
}

func TestCleanupDelete(t *testing.T) {
	mockClient := DummyHttp{
		responseBodies: []string{
			// SOA serial of 2022040507.
			// TXT record of dummy/test-value-other, line 17
			// TXT record of dummy/test-value, line 18
			`{
				"data": [
					{
						"line_index": 3,
						"type": "record",
						"data_b64": [
							"bnMxLnN0YWJsZWhvc3QuY29tLg==",
							"YWxlcnRzLnN0YWJsZWhvc3QuY29tLg==",
							"MjAyMjA0MDUwNQ==",
							"ODY0MDA=",
							"NzIwMA==",
							"MzYwMDAwMA==",
							"MTgwMA=="
						],
						"dname_b64": "amFtZXNsYWtpbi5jby51ay4=",
						"record_type": "SOA",
						"ttl": 86400
					},
					{
						"line_index": 17,
						"type": "record",
						"data_b64": [
							"dGVzdC12YWx1ZS1vdGhlcg=="
						],
						"dname_b64": "ZHVtbXk=",
						"record_type": "TXT",
						"ttl": 1
					},
					{
						"line_index": 18,
						"type": "record",
						"data_b64": [
							"dGVzdC12YWx1ZQ=="
						],
						"dname_b64": "ZHVtbXk=",
						"record_type": "TXT",
						"ttl": 1
					}
				]
			}`,
			`{}`, // Errors needs to be empty, that's all
		},
	}
	client := NewClientWithMock(&mockClient)

	err := client.ClearDnsTxt("dummy.test-domain.com.", "test-value")
	assert.NoError(t, err)

	// Expect 2 requests (zone info, delete)
	assert.Len(t, mockClient.requests, 2)

	// Delete
	request := mockClient.requests[1]
	expectedCreateUrl := `https://cpanel.test-domain.com/execute/DNS/mass_edit_zone?zone=test-domain.com&serial=2022040505&remove=18`
	assert.Equal(t, expectedCreateUrl, request.URL.String())
	assert.Equal(t, expectedUsernamePasswordAuthorization, request.Header["Authorization"][0])
}

func TestCleanupNoDelete(t *testing.T) {
	mockClient := DummyHttp{
		responseBodies: []string{
			// SOA serial of 2022040507.
			`{
				"data": [
					{
						"line_index": 3,
						"type": "record",
						"data_b64": [
							"bnMxLnN0YWJsZWhvc3QuY29tLg==",
							"YWxlcnRzLnN0YWJsZWhvc3QuY29tLg==",
							"MjAyMjA0MDUwNQ==",
							"ODY0MDA=",
							"NzIwMA==",
							"MzYwMDAwMA==",
							"MTgwMA=="
						],
						"dname_b64": "amFtZXNsYWtpbi5jby51ay4=",
						"record_type": "SOA",
						"ttl": 86400
					}
				]
			}`,
		},
	}
	client := NewClientWithMock(&mockClient)

	err := client.ClearDnsTxt("dummy.test-domain.com.", "test-value")
	assert.NoError(t, err)

	// Expect 1 request, only zone info
	assert.Len(t, mockClient.requests, 1)
	assert.Equal(t, expectedUsernamePasswordAuthorization, mockClient.requests[0].Header["Authorization"][0])
}
