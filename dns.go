package porkbun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const PORKBUN_DNS_BASE = PORKBUN_API_BASE + "/dns"
const PORKBUN_DNS_CREATE = PORKBUN_DNS_BASE + "/create/%s"
const PORKBUN_DNS_EDIT = PORKBUN_DNS_BASE + "/edit/%s/%s"
const PORKBUN_DNS_DELETE = PORKBUN_DNS_BASE + "/delete/%s/%s"
const PORKBUN_DNS_RETRIEVE = PORKBUN_DNS_BASE + "/retrieve/%s"
const STATUS_SUCCESS = "SUCCESS"

type Client struct {
	config Config
}

type Config struct {
	Auth   Auth
	Client *http.Client
}

type Auth struct {
	APIKey       string `json:"apikey,omitempty"`
	SecretAPIKey string `json:"secretapikey,omitempty"`
}

type DNSRecord struct {
	ID      string  `json:"id,omitempty"`
	Name    string  `json:"name,omitempty"`
	Type    string  `json:"type,omitempty"`
	Content string  `json:"content,omitempty"`
	TTL     int     `json:"ttl,omitempty"`
	Prio    *uint16 `json:"prio,omitempty"`
	Notes   string  `json:"notes,omitempty"`
}

type DNSResponse struct {
	Status  string      `json:"status,omitempty"`
	Id      string      `json:"id,omitempty"`
	Records []DNSRecord `json:"records,omitempty"`
}

type dnsRecordWithAuth struct {
	Auth
	DNSRecord
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg.Auth.APIKey == "" {
		return nil, fmt.Errorf("APIKey should not be empty")
	}
	if cfg.Auth.SecretAPIKey == "" {
		return nil, fmt.Errorf("SecretAPIKey should not be empty")
	}
	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}
	return &Client{config: *cfg}, nil
}

func (c *Client) getAuthJson() ([]byte, error) {
	json, err := json.Marshal(c.config.Auth)
	if err != nil {
		return nil, fmt.Errorf("Error creating json")
	}
	return json, nil
}

func (c *Client) getDNSRecordWithAuthJson(dnsRecord *DNSRecord) ([]byte, error) {
	lee := dnsRecordWithAuth{
		Auth:      c.config.Auth,
		DNSRecord: *dnsRecord,
	}
	json, err := json.Marshal(lee)
	if err != nil {
		return nil, fmt.Errorf("Error creating json")
	}
	return json, nil
}

// Helper land
func requireSuccess(dnsRes *DNSResponse) error {
	if !strings.EqualFold(dnsRes.Status, STATUS_SUCCESS) {
		return fmt.Errorf("Expected `success` code, got %s", dnsRes.Status)
	}
	return nil
}

func requireOK(res *http.Response, err error) (*http.Response, error) {
	if err != nil {
		if res != nil {
			res.Body.Close()
		}
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, generateUnexpectedResponseCodeError(res)
	}
	return res, nil
}

func generateUnexpectedResponseCodeError(resp *http.Response) error {
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)
	resp.Body.Close()
	return fmt.Errorf("Unexpected response code: %d (%s)", resp.StatusCode, buf.Bytes())
}

func extractDNSResponse(res *http.Response, err error) (*DNSResponse, error) {
	if err != nil {
		return &DNSResponse{}, err
	}
	var dnsResp DNSResponse
	if err := json.NewDecoder(res.Body).Decode(&dnsResp); err != nil {
		return &DNSResponse{}, fmt.Errorf("Error decoding DNSResponse json")
	}
	if requireSuccess(&dnsResp) != nil {
		return &DNSResponse{}, err
	}
	return &dnsResp, nil
}

// Main function land
func (c *Client) CreateRecord(domain string, dnsrecord *DNSRecord) (*DNSResponse, error) {
	authjson, err := c.getDNSRecordWithAuthJson(dnsrecord)
	if err != nil {
		return &DNSResponse{}, err
	}
	res, err := requireOK(
		c.config.Client.Post(
			fmt.Sprintf(PORKBUN_DNS_CREATE, domain),
			"application/json",
			bytes.NewBuffer(authjson)),
	)
	defer res.Body.Close()
	return extractDNSResponse(res, err)
}

func (c *Client) EditRecord(domain string, id string, dnsrecord *DNSRecord) (*DNSResponse, error) {
	authjson, err := c.getDNSRecordWithAuthJson(dnsrecord)
	if err != nil {
		return &DNSResponse{}, err
	}
	res, err := requireOK(
		c.config.Client.Post(
			fmt.Sprintf(PORKBUN_DNS_EDIT, domain, id),
			"application/json",
			bytes.NewBuffer(authjson)),
	)
	defer res.Body.Close()
	return extractDNSResponse(res, err)
}

func (c *Client) DeleteRecord(domain string, id string) (*DNSResponse, error) {
	authjson, err := c.getAuthJson()
	if err != nil {
		return &DNSResponse{}, err
	}
	res, err := requireOK(
		c.config.Client.Post(
			fmt.Sprintf(PORKBUN_DNS_DELETE, domain, id),
			"application/json",
			bytes.NewBuffer(authjson)),
	)
	defer res.Body.Close()
	return extractDNSResponse(res, err)
}

func (c *Client) RetrieveRecords(domain string) (*DNSResponse, error) {
	authjson, err := c.getAuthJson()
	if err != nil {
		return &DNSResponse{}, err
	}
	res, err := requireOK(
		c.config.Client.Post(
			fmt.Sprintf(PORKBUN_DNS_RETRIEVE, domain),
			"application/json",
			bytes.NewBuffer(authjson)),
	)
	defer res.Body.Close()
	return extractDNSResponse(res, err)
}
