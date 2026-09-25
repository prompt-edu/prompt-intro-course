package appleTeam

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const apiBase = "https://api.appstoreconnect.apple.com"

type Client struct {
	issuer string
	keyID  string
	key    *ecdsa.PrivateKey
	http   *http.Client
}

type Capacity struct {
	RegisteredIPhones int `json:"registeredIPhones"`
	AvailableIPhones  int `json:"availableIPhones"`
	Limit             int `json:"limit"`
}

type AccountStatus struct {
	Membership          string          `json:"membership"`
	ProvisioningAllowed bool            `json:"provisioningAllowed"`
	Devices             map[string]bool `json:"devices"`
}

type applePage struct {
	Data []struct {
		Attributes struct {
			DeviceClass         string `json:"deviceClass"`
			Status              string `json:"status"`
			UDID                string `json:"udid"`
			Username            string `json:"username"`
			Email               string `json:"email"`
			ProvisioningAllowed bool   `json:"provisioningAllowed"`
		} `json:"attributes"`
	} `json:"data"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
}

func Configured() bool {
	return os.Getenv("APPLE_ISSUER_ID") != "" && os.Getenv("APPLE_KEY_ID") != "" && os.Getenv("APPLE_PRIVATE_KEY_B64") != ""
}

func NewFromEnvironment() (*Client, error) {
	if !Configured() {
		return nil, errors.New("Apple team API credentials are not configured")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(os.Getenv("APPLE_PRIVATE_KEY_B64"))
	if err != nil {
		return nil, errors.New("Apple team private key is not valid base64")
	}
	keyBytes = []byte(strings.ReplaceAll(string(keyBytes), `\n`, "\n"))
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("Apple team private key is not valid PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("Apple team private key is not valid PKCS8")
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("Apple team private key is not an EC key")
	}
	return &Client{
		issuer: os.Getenv("APPLE_ISSUER_ID"), keyID: os.Getenv("APPLE_KEY_ID"), key: key,
		http: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
	}, nil
}

func encodeJSON(value any) string {
	data, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(data)
}

func (c *Client) token() (string, error) {
	now := time.Now().Unix()
	unsigned := encodeJSON(map[string]any{"alg": "ES256", "kid": c.keyID, "typ": "JWT"}) + "." +
		encodeJSON(map[string]any{"iss": c.issuer, "iat": now, "exp": now + 600, "aud": "appstoreconnect-v1"})
	hash := sha256.Sum256([]byte(unsigned))
	r, s, err := ecdsa.Sign(rand.Reader, c.key, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign Apple API request: %w", err)
	}
	signature := append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (c *Client) page(ctx context.Context, path string) (*applePage, error) {
	endpoint, err := url.Parse(path)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host != "api.appstoreconnect.apple.com" {
		return nil, errors.New("Apple API returned an invalid page URL")
	}
	token, err := c.token()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query Apple team API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple team API returned HTTP %d", resp.StatusCode)
	}
	var result applePage
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil {
		return nil, errors.New("Apple team API returned an invalid response")
	}
	return &result, nil
}

func (c *Client) Capacity(ctx context.Context) (Capacity, error) {
	result := Capacity{Limit: 100}
	path := apiBase + "/v1/devices?limit=200&fields%5Bdevices%5D=deviceClass%2Cstatus"
	for path != "" {
		page, err := c.page(ctx, path)
		if err != nil {
			return Capacity{}, err
		}
		for _, device := range page.Data {
			// Disabled devices still consume a slot until the membership-year reset.
			if device.Attributes.DeviceClass == "IPHONE" {
				result.RegisteredIPhones++
			}
		}
		path = page.Links.Next
	}
	result.AvailableIPhones = max(0, result.Limit-result.RegisteredIPhones)
	return result, nil
}

func (c *Client) lookup(ctx context.Context, resource, filter, value, field string) (*applePage, error) {
	query := url.Values{}
	query.Set("filter["+filter+"]", value)
	query.Set("fields["+resource+"]", field)
	query.Set("limit", "200")
	return c.page(ctx, apiBase+"/v1/"+resource+"?"+query.Encode())
}

func (c *Client) ProfileStatus(ctx context.Context, email string, devices map[string]string) (AccountStatus, error) {
	status := AccountStatus{Membership: "none", Devices: map[string]bool{}}
	if email != "" {
		users, err := c.lookup(ctx, "users", "username", email, "username,provisioningAllowed")
		if err != nil {
			return status, err
		}
		for _, user := range users.Data {
			if strings.EqualFold(user.Attributes.Username, email) {
				status.Membership = "active"
				status.ProvisioningAllowed = user.Attributes.ProvisioningAllowed
				break
			}
		}
		if status.Membership == "none" {
			invitations, err := c.lookup(ctx, "userInvitations", "email", email, "email,provisioningAllowed")
			if err != nil {
				return status, err
			}
			for _, invitation := range invitations.Data {
				if strings.EqualFold(invitation.Attributes.Email, email) {
					status.Membership = "invited"
					status.ProvisioningAllowed = invitation.Attributes.ProvisioningAllowed
					break
				}
			}
		}
	}
	for kind, udid := range devices {
		if udid == "" {
			continue
		}
		status.Devices[kind] = false
		matches, err := c.lookup(ctx, "devices", "udid", udid, "udid,status")
		if err != nil {
			return status, err
		}
		for _, device := range matches.Data {
			if strings.EqualFold(device.Attributes.UDID, udid) && device.Attributes.Status == "ENABLED" {
				status.Devices[kind] = true
			}
		}
	}
	return status, nil
}
