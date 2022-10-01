package cpanel

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"strings"

	log "github.com/sirupsen/logrus"
)

type CpanelClient struct {
	httpClient http.Client
	DnsZone    string
	CpanelUrl  string
	Username   string
	Password   string
	ApiToken   string // An alternative to a password and takes precedence
}

func (c *CpanelClient) SetDnsTxt(recordName string, value string) error {
	log.Infof("Setting TXT record for '%s' to '%s'", recordName, value)
	recordNameSub := c.getDnsSubdomainOnly(recordName)

	zone, err := c.getZoneDetails()
	if err != nil {
		return err
	}
	log.Infof("Got zone, record count: %d", len(zone.Data))

	// Get the zone serial as it's needed for mutation
	serial := getZoneSerial(zone)
	log.Infof("Got SOA serial %s", serial)

	// Does the requested record already exist?
	var existingRecord *cpanelZoneRecord

	// All records of a given key must have the same TTL.
	// If other ACME clients have set DNS records we need to grab the TTL and use that, else we default to 300.
	existingRecordTtl := 300
	for _, record := range zone.Data {
		if record.RecordType == typeTxt && record.Dname == recordNameSub {
			existingRecordTtl = record.TTL
			// Found a record, but does it have the right value? (there could be multiple TXTs)
			if len(record.Data) > 0 && record.Data[0] == value {
				existingRecord = &record
				break
			} else {
				log.Debugf("Found an existing record but had different value. TTL: %d", existingRecordTtl)
			}
		}
	}

	if existingRecord != nil {
		log.Info("Existing record with matching value found, not doing anything")
	} else {
		log.Info("No existing record with value exists, creating it")
		err := c.createZoneRecord(serial, recordNameSub, value, existingRecordTtl)
		if err == nil {
			log.Info("Record created")
		} else {
			log.Error("Could not create record", err)
		}
		return err
	}

	return nil
}

func (c *CpanelClient) ClearDnsTxt(recordName string, value string) error {
	log.Infof("Deleting TXT record for '%s' and value '%s'", recordName, value)
	recordNameSub := c.getDnsSubdomainOnly(recordName)

	log.Debug("Getting zone")
	zone, err := c.getZoneDetails()
	if err != nil {
		return err
	}
	log.Debug("Got zone")

	// Get the zone serial as it's needed for mutation
	serial := getZoneSerial(zone)
	log.Infof("Got SOA serial %s", serial)

	var existingRecord *cpanelZoneRecord
	for _, record := range zone.Data {
		if record.RecordType == typeTxt && record.Dname == recordNameSub {
			if len(record.Data) > 0 && record.Data[0] == value {
				existingRecord = &record
			}
		}
	}

	if existingRecord == nil {
		log.Warn("Record not found - has it already been deleted? Pretending it was successful")
		return nil
	} else {
		log.Debugf("Record found with line no %d", existingRecord.LineIndex)
		err := c.deleteZoneRecord(serial, existingRecord.LineIndex)
		return err
	}
}

func (c *CpanelClient) getZoneDetails() (*cpanelZoneResponse, error) {
	req, err := http.NewRequest("GET", c.CpanelUrl+"/execute/DNS/parse_zone?zone="+url.QueryEscape(c.getDnsZoneNoDot()), nil)
	if err != nil {
		log.Error("zone info HTTP request error", err)
		return nil, err
	}

	c.addRequestAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("zone info HTTP response error", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("zone info HTTP response read error", err)
		return nil, err
	}

	var zoneResponse cpanelZoneResponse
	err = json.Unmarshal(bodyBytes, &zoneResponse)
	if err != nil {
		log.Error("could not decode zone JSON", err)
		return nil, err
	}

	if len(zoneResponse.Errors) > 0 {
		log.Errorf("zone JSON reported errors: %+v", zoneResponse.Errors)
		return &zoneResponse, errors.New("zone JSON reported errors")
	}

	// The CPanel API unhelpfully base64 encodes values
	dataRecords := zoneResponse.Data
	for i := range dataRecords {
		record := &dataRecords[i]
		record.Dname = decode(record.DnameB64)
		record.Text = decode(record.TextB64)

		for _, data := range record.DataB64 {
			dataValue := decode(data)
			record.Data = append(record.Data, dataValue)
		}
	}

	return &zoneResponse, nil
}

func (c *CpanelClient) createZoneRecord(serial string, recordName string, value string, ttl int) error {
	// TODO: URL encode
	createObj := &cpanelZoneRecordAdd{
		Data:       []string{value},
		Dname:      recordName,
		TTL:        ttl,
		RecordType: typeTxt,
	}

	createJson, err := json.Marshal(createObj)
	createJsonEncoded := url.QueryEscape(string(createJson))
	if err != nil {
		log.Error("could not marshal JSON for create", err)
		return err
	}

	url := c.CpanelUrl + "/execute/DNS/mass_edit_zone?zone=" + c.getDnsZoneNoDot() + "&serial=" + serial + "&add=" + createJsonEncoded
	log.Debugf("Using URL to create: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("zone cretae HTTP request error", err)
		return err
	}

	c.addRequestAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("zone create HTTP response error", err)
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("zone create HTTP response read error", err)
		return err
	}

	var createResponse cpanelResponse
	err = json.Unmarshal(bodyBytes, &createResponse)
	if err != nil {
		log.Error("could not decode create JSON", err)
		return err
	}

	if len(createResponse.Errors) > 0 {
		log.Errorf("create JSON reported errors: %+v", createResponse.Errors)
		return errors.New("create JSON reported errors")
	}

	return nil
}

func (c *CpanelClient) deleteZoneRecord(serial string, recordLineNo int) error {
	url := c.CpanelUrl + "/execute/DNS/mass_edit_zone?zone=" + c.getDnsZoneNoDot() + "&serial=" + serial + "&remove=" + strconv.Itoa(recordLineNo)
	log.Debugf("Using URL to delete: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("zone delete HTTP request error", err)
		return err
	}

	c.addRequestAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("zone delete HTTP response error", err)
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("zone delete HTTP response read error", err)
		return err
	}

	var createResponse cpanelResponse
	err = json.Unmarshal(bodyBytes, &createResponse)
	if err != nil {
		log.Error("could not decode delete JSON", err)
		return err
	}

	if len(createResponse.Errors) > 0 {
		log.Errorf("delete JSON reported errors: %+v", createResponse.Errors)
		return errors.New("delete JSON reported errors")
	}

	return nil
}

func (c *CpanelClient) getDnsSubdomainOnly(recordName string) string {
	// recordName will be in the form 'my-subdomain.my-domain.com.'
	// We need to strip out the zone ('.my-domain.com.') from it.
	recordNameSub := strings.TrimSuffix(recordName, "."+c.DnsZone)
	log.Debugf("Calculated record name '%s' for zone '%s'", recordNameSub, c.DnsZone)
	return recordNameSub
}

func (c *CpanelClient) getDnsZoneNoDot() string {
	// CPanel API expects zone of 'my-domain.com', not 'my-domain.com.'
	return strings.TrimSuffix(c.DnsZone, ".")
}

// Add either Basic auth for username/password or CPanel's own API Token mechanism
func (c *CpanelClient) addRequestAuth(req *http.Request) {
	if c.ApiToken != "" {
		req.Header.Add("Authorization", "cpanel "+c.Username+":"+c.ApiToken)
	} else {
		req.SetBasicAuth(c.Username, c.Password)
	}
}

// ----
// Utils
// ----

// Get the SOA serial needed for mutating DNS zones.
func getZoneSerial(zone *cpanelZoneResponse) string {
	for _, record := range zone.Data {
		if record.RecordType == typeSoa {
			return record.Data[soaSerialIndex]
		}
	}

	return "" // Not found!
}

func decode(b64 string) string {
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return ""
	}

	return string(decoded)
}

// ----
// Types
// ----

type recordType string

const typeSoa recordType = "SOA"
const typeTxt recordType = "TXT"

const soaSerialIndex = 2

type cpanelResponse struct {
	Errors []string `json:"errors"`
	Status int      `json:"status"`
}

// https://api.docs.cpanel.net/openapi/cpanel/operation/dns-parse_zone/
type cpanelZoneResponse struct {
	cpanelResponse
	Data []cpanelZoneRecord `json:"data"`
}

type cpanelZoneRecord struct {
	LineIndex int    `json:"line_index"`
	Type      string `json:"type"`
	// Most values have 1 element, but others like SOA have many
	DataB64    []string   `json:"data_b64"`
	DnameB64   string     `json:"dname_b64"`
	RecordType recordType `json:"record_type"`
	TTL        int        `json:"ttl"`
	TextB64    string     `json:"text_b64"`

	Data  []string
	Dname string
	Text  string
}

// https://api.docs.cpanel.net/openapi/cpanel/operation/dns-mass_edit_zone/
type cpanelZoneRecordAdd struct {
	Dname      string     `json:"dname"`
	TTL        int        `json:"ttl"`
	RecordType recordType `json:"record_type"`
	Data       []string   `json:"data"`
}
