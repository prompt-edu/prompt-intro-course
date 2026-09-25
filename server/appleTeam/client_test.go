package appleTeam

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (fn transportFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestAppleTeamTokenAndCapacity(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{issuer: "issuer", keyID: "KEY1234567", key: key}
	client.http = &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.appstoreconnect.apple.com" || req.URL.Path != "/v1/devices" {
			t.Fatalf("unexpected Apple API URL: %s", req.URL)
		}
		token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			t.Fatal("invalid JWT")
		}
		var claims map[string]any
		payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
		if err := json.Unmarshal(payload, &claims); err != nil || claims["iss"] != "issuer" || claims["aud"] != "appstoreconnect-v1" {
			t.Fatal("incorrect Apple API claims")
		}
		signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
		if len(signature) != 64 {
			t.Fatal("invalid ES256 signature length")
		}
		hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
		if !ecdsa.Verify(&key.PublicKey, hash[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
			t.Fatal("Apple API signature did not verify")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"attributes":{"deviceClass":"IPHONE","status":"ENABLED"}},{"attributes":{"deviceClass":"IPHONE","status":"DISABLED"}},{"attributes":{"deviceClass":"IPAD","status":"ENABLED"}},{"attributes":{"deviceClass":"APPLE_WATCH","status":"ENABLED"}}],"links":{"next":""}}`))}, nil
	})}
	capacity, err := client.Capacity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if capacity.RegisteredIPhones != 2 || capacity.AvailableIPhones != 98 || capacity.AvailableIPads != 99 || capacity.AvailableWatches != 99 {
		t.Fatalf("incorrect capacity: %+v", capacity)
	}
}

func TestProfileStatusChecksOnlyRequestedAccountAndDevice(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{issuer: "issuer", keyID: "KEY1234567", key: key}
	client.http = &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/v1/users":
			if req.URL.Query().Get("filter[username]") != "student@example.edu" {
				t.Fatal("unexpected account lookup")
			}
			body = `{"data":[{"attributes":{"username":"student@example.edu","provisioningAllowed":true}}]}`
		case "/v1/devices":
			if req.URL.Query().Get("filter[udid]") != "TEST-UDID" {
				t.Fatal("unexpected device lookup")
			}
			body = `{"data":[{"attributes":{"udid":"TEST-UDID","status":"ENABLED"}}]}`
		default:
			t.Fatalf("unexpected Apple API path: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	status, err := client.ProfileStatus(context.Background(), "student@example.edu", map[string]string{"iPhone": "TEST-UDID"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Membership != "active" || !status.ProvisioningAllowed || !status.Devices["iPhone"] {
		t.Fatalf("incorrect Apple profile status: %+v", status)
	}
}

func TestInviteDeveloperUsesSavedIdentityAndProvisioning(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{issuer: "issuer", keyID: "KEY1234567", key: key}
	posts := 0
	client.http = &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/v1/users", "/v1/userInvitations":
			if req.Method == http.MethodGet {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))}, nil
			}
			posts++
			var request struct {
				Data struct {
					Type       string `json:"type"`
					Attributes struct {
						Email               string   `json:"email"`
						FirstName           string   `json:"firstName"`
						LastName            string   `json:"lastName"`
						Roles               []string `json:"roles"`
						ProvisioningAllowed bool     `json:"provisioningAllowed"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Data.Type != "userInvitations" || request.Data.Attributes.Email != "student@example.edu" ||
				request.Data.Attributes.FirstName != "First" || request.Data.Attributes.LastName != "Last" ||
				len(request.Data.Attributes.Roles) != 1 || request.Data.Attributes.Roles[0] != "DEVELOPER" ||
				!request.Data.Attributes.ProvisioningAllowed {
				t.Fatalf("incorrect invitation request: %+v", request)
			}
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}
		t.Fatalf("unexpected path: %s", req.URL.Path)
		return nil, nil
	})}
	created, err := client.InviteDeveloper(context.Background(), "student@example.edu", "First", "Last")
	if err != nil || !created || posts != 1 {
		t.Fatalf("invitation failed: created=%v posts=%d err=%v", created, posts, err)
	}
}

func TestRegisterDeviceChecksCapacityBeforeWriting(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{issuer: "issuer", keyID: "KEY1234567", key: key}
	posts := 0
	client.http = &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/devices" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if req.Method == http.MethodPost {
			posts++
			var request struct {
				Data struct {
					Type       string `json:"type"`
					Attributes struct {
						Name     string `json:"name"`
						Platform string `json:"platform"`
						UDID     string `json:"udid"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Data.Type != "devices" || request.Data.Attributes.Name != "Intro Course Device" ||
				request.Data.Attributes.Platform != "IOS" || request.Data.Attributes.UDID != "TEST-UDID" {
				t.Fatalf("incorrect device request: %+v", request)
			}
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}
		if req.URL.Query().Has("filter[udid]") {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[],"links":{"next":""}}`))}, nil
	})}
	created, err := client.RegisterDevice(context.Background(), "TEST-UDID", "Intro Course Device", "iphone")
	if err != nil || !created || posts != 1 {
		t.Fatalf("registration failed: created=%v posts=%d err=%v", created, posts, err)
	}
}
