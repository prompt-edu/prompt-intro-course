package appleTeam

import (
	"bytes"
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
	RegisteredIPads   int `json:"registeredIPads"`
	AvailableIPads    int `json:"availableIPads"`
	RegisteredWatches int `json:"registeredWatches"`
	AvailableWatches  int `json:"availableWatches"`
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
	return strings.TrimSpace(os.Getenv("APPLE_ISSUER_ID")) != "" && strings.TrimSpace(os.Getenv("APPLE_KEY_ID")) != "" && os.Getenv("APPLE_PRIVATE_KEY_B64") != ""
}

func NewFromEnvironment() (*Client, error) {
	if !Configured() {
		return nil, errors.New("apple team API credentials are not configured")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(os.Getenv("APPLE_PRIVATE_KEY_B64"))
	if err != nil {
		return nil, errors.New("apple team private key is not valid base64")
	}
	keyBytes = []byte(strings.ReplaceAll(string(keyBytes), `\n`, "\n"))
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("apple team private key is not valid PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("apple team private key is not valid PKCS8")
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("apple team private key is not an EC key")
	}
	return &Client{
		issuer: strings.TrimSpace(os.Getenv("APPLE_ISSUER_ID")), keyID: strings.TrimSpace(os.Getenv("APPLE_KEY_ID")), key: key,
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
		return nil, errors.New("apple API returned an invalid page URL")
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
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apple team API returned HTTP %d", resp.StatusCode)
	}
	var result applePage
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil {
		return nil, errors.New("apple team API returned an invalid response")
	}
	return &result, nil
}

func (c *Client) create(ctx context.Context, resource string, attributes any) error {
	token, err := c.token()
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{"data": map[string]any{"type": resource, "attributes": attributes}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/v1/"+resource, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("contact Apple team API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Apple team API rejected %s with HTTP %d", resource, resp.StatusCode)
	}
	return nil
}

// InviteDeveloper grants provisioning access. A pending or active invitation is
// left alone so a retry never sends a second email.
func (c *Client) InviteDeveloper(ctx context.Context, email, firstName, lastName string) (bool, error) {
	status, err := c.ProfileStatus(ctx, email, nil)
	if err != nil {
		return false, err
	}
	if status.Membership != "none" {
		if !status.ProvisioningAllowed {
			return false, errors.New("existing Apple team access lacks provisioning; update that membership in App Store Connect")
		}
		return false, nil
	}
	if err := c.create(ctx, "userInvitations", map[string]any{
		"email": email, "firstName": firstName, "lastName": lastName,
		"roles": []string{"DEVELOPER"}, "provisioningAllowed": true,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// RegisterDevice is idempotent for an enabled UDID. Apple does not restore a
// disabled device through this endpoint; the lecturer must resolve that case.
func (c *Client) RegisterDevice(ctx context.Context, udid, name, kind string) (bool, error) {
	matches, err := c.lookup(ctx, "devices", "udid", udid, "udid,status")
	if err != nil {
		return false, err
	}
	for _, device := range matches.Data {
		if strings.EqualFold(device.Attributes.UDID, udid) {
			if device.Attributes.Status == "ENABLED" {
				return false, nil
			}
			return false, errors.New("this device is disabled in Apple Developer; resolve it there")
		}
	}
	capacity, err := c.Capacity(ctx)
	if err != nil {
		return false, err
	}
	available := map[string]int{
		"iphone": capacity.AvailableIPhones,
		"ipad":   capacity.AvailableIPads,
		"watch":  capacity.AvailableWatches,
	}[kind]
	if available < 1 {
		return false, fmt.Errorf("no Apple %s registration slots remain", kind)
	}
	if err := c.create(ctx, "devices", map[string]any{
		"name": name, "platform": "IOS", "udid": udid,
	}); err != nil {
		return false, err
	}
	return true, nil
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
			switch device.Attributes.DeviceClass {
			case "IPHONE":
				result.RegisteredIPhones++
			case "IPAD":
				result.RegisteredIPads++
			case "APPLE_WATCH":
				result.RegisteredWatches++
			}
		}
		path = page.Links.Next
	}
	result.AvailableIPhones = max(0, result.Limit-result.RegisteredIPhones)
	result.AvailableIPads = max(0, result.Limit-result.RegisteredIPads)
	result.AvailableWatches = max(0, result.Limit-result.RegisteredWatches)
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
