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

	for k, v := range secrets {
		os.Setenv(strings.ToUpper(k), v)
	}

	var origCmd []string
	var origArgs []string
	if v := os.Getenv("VAULT_ORIG_CMD"); v != "" {
		_ = json.Unmarshal([]byte(v), &origCmd)
	}
	if v := os.Getenv("VAULT_ORIG_ARGS"); v != "" {
		_ = json.Unmarshal([]byte(v), &origArgs)
	}

	if len(origCmd) == 0 {
		origCmd = []string{"/bin/sh"}
		origArgs = []string{"-c", os.Getenv("VAULT_APP_CMD")}
	}

	cmd := exec.Command(origCmd[0], append(origCmd[1:], origArgs...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Fatalf("exec failed: %v", err)
	}
}

type config struct {
	Addr       string
	Path       string
	SecretKeys []string
	Role       string
	Namespace  string
	Insecure   bool
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
		Namespace:  os.Getenv("VAULT_NAMESPACE"),
		Insecure:   insecure,
	}
}

type vaultClient struct {
	addr      string
	namespace string
	role      string
	token     string
	insecure  bool
	httpc     *http.Client
}

func newVaultClient(cfg config) *vaultClient {
	tr := &http.Transport{}
	if cfg.Insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &vaultClient{
		addr:      cfg.Addr,
		namespace: cfg.Namespace,
		role:      cfg.Role,
		insecure:  cfg.Insecure,
		httpc:     &http.Client{Timeout: 10 * time.Second, Transport: tr},
	}
}

func (v *vaultClient) loginWithKubernetes() (string, error) {
	jwt, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("read jwt: %w", err)
	}

	reqBody := map[string]string{"role": v.role, "jwt": string(jwt)}
	bs, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal login body: %w", err)
	}

	url := strings.TrimRight(v.addr, "/") + "/v1/auth/kubernetes/login"
	req, err := http.NewRequest("POST", url, strings.NewReader(string(bs)))
	if err != nil {
		return "", fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpc.Do(req)
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

func (v *vaultClient) fetchKVv2Secret(path string, secretKeys []string) (map[string]string, error) {
	res := map[string]string{}
	url := strings.TrimRight(v.addr, "/") + "/v1/" + strings.TrimPrefix(path, "/")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create secret request: %w", err)
	}
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

	if d, ok := body["data"].(map[string]interface{}); ok {
		if inner, ok := d["data"].(map[string]interface{}); ok {
			if len(secretKeys) == 0 {
				for k, val := range inner {
					res[k] = fmt.Sprintf("%v", val)
				}
			} else {
				for _, key := range secretKeys {
					if val, exists := inner[key]; exists {
						res[key] = fmt.Sprintf("%v", val)
					}
				}
			}
		}
	}
	return res, nil
}

var tlsConfigInsecure = tls.Config{InsecureSkipVerify: true}
