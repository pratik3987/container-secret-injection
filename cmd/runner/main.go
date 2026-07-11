package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	cfg := loadConfigFromEnv()
	client := newVaultClient(cfg)

	token, err := client.loginWithKubernetes()
	if err != nil {
		log.Fatalf("vault login failed: %v", err)
	}
	client.token = token

	secrets, err := client.fetchKVv2Secret(cfg.Path, cfg.SecretKeys)
	if err != nil {
		log.Fatalf("fetch secrets: %v", err)
	}

	// export as env
	for k, v := range secrets {
		os.Setenv(strings.ToUpper(k), v)
	}

	// get original command
	var origCmd []string
	var origArgs []string
	if v := os.Getenv("VAULT_ORIG_CMD"); v != "" {
		_ = json.Unmarshal([]byte(v), &origCmd)
	}
	if v := os.Getenv("VAULT_ORIG_ARGS"); v != "" {
		_ = json.Unmarshal([]byte(v), &origArgs)
	}

	// if origCmd is empty, try to read from VAULT_APP_IMAGE or fallback to sh -c
	if len(origCmd) == 0 {
		origCmd = []string{"/bin/sh"}
		origArgs = []string{"-c", os.Getenv("VAULT_APP_CMD")}
	}

	// exec
	cmd := exec.Command(origCmd[0], append(origCmd[1:], origArgs...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Fatalf("exec failed: %v", err)
	}
}

type config struct {
	Addr         string
	Path         string
	SecretKeys   []string
	Role         string
	Insecure     bool
}

func loadConfigFromEnv() config {
	var keys []string
	if s := os.Getenv("VAULT_SECRET_KEYS"); s != "" {
		for _, k := range strings.Split(s, ",") {
			keys = append(keys, strings.TrimSpace(k))
		}
	}
	insecure := false
	if os.Getenv("VAULT_INSECURE") == "true" {
		insecure = true
	}
	return config{
		Addr:       os.Getenv("VAULT_ADDR"),
		Path:       os.Getenv("VAULT_PATH"),
		SecretKeys: keys,
		Role:       os.Getenv("VAULT_ROLE"),
		Insecure:   insecure,
	}
}

// minimal Vault client wrapper
type vaultClient struct {
	addr     string
	role     string
	token    string
	insecure bool
	httpc    *http.Client
}

func newVaultClient(cfg config) *vaultClient {
	tr := &http.Transport{}
	if cfg.Insecure {
		tr.TLSClientConfig = &tlsConfigInsecure
	}
	return &vaultClient{addr: cfg.Addr, namespace: cfg.Namespace, role: cfg.Role, insecure: cfg.Insecure, httpc: &http.Client{Timeout: 10 * time.Second, Transport: tr}}
}

// loginWithKubernetes reads the service account JWT and performs Kubernetes auth
func (v *vaultClient) loginWithKubernetes() (string, error) {
	jwt, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("read jwt: %w", err)
	}
	reqBody := map[string]string{"role": v.role, "jwt": string(jwt)}
	bs, _ := json.Marshal(reqBody)
	url := strings.TrimRight(v.addr, "/") + "/v1/auth/kubernetes/login"
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(bs)))
	req.Header.Set("Content-Type", "application/json")
	if v.namespace != "" {esp, err := v.httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Auth.ClientToken, nil
}

// fetchKVv2Paths fetches secrets from provided kv v2 paths and returns merged map
func (v *vaultClient) fetchKVv2Paths(paths []string) (map[string]string, error) {
	res := map[string]string{}
	for _, p := range paths {
		url := strSecret fetches secrets from a KV v2 path
// If secretKeys is empty, fetches all keys from the path
// If secretKeys is provided, returns only those keys
func (v *vaultClient) fetchKVv2Secret(path string, secretKeys []string) (map[string]string, error) {
	res := map[string]string{}
	
	url := strings.TrimRight(v.addr, "/") + "/v1/" + strings.TrimPrefix(path, "/")
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Vault-Token", v.token)
	
	resp, err := v.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	
	// KV v2 returns {data:{data: {...}}}
	if d, ok := body["data"].(map[string]interface{}); ok {
		if inner, ok := d["data"].(map[string]interface{}); ok {
			// If no specific keys requested, fetch all
			if len(secretKeys) == 0 {
				for k, v := range inner {
					res[k] = fmt.Sprintf("%v", v)
				}
			} else {
				// Otherwise, fetch only requested keys
				for _, key := range secretKeys {
					if val, exists := inner[key]; exists {
						res[key] = fmt.Sprintf("%v", val)
					}
	return res, nil
}

// minimal insecure TLS config placeholder
var tlsConfigInsecure = tls.Config{InsecureSkipVerify: true}
