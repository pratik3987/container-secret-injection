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

    secrets, err := client.fetchKVv2Paths(cfg.SecretPaths)
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
    Addr        string
    Namespace   string
    Role        string
    AuthMethod  string
    SecretPaths []string
    Insecure    bool
}

func loadConfigFromEnv() config {
    paths := []string{}
    if s := os.Getenv("VAULT_SECRET_PATHS"); s != "" {
        for _, p := range strings.Split(s, ",") {
            paths = append(paths, strings.TrimSpace(p))
        }
    }
    insecure := false
    if os.Getenv("VAULT_INSECURE_SKIP_VERIFY") == "true" {
        insecure = true
    }
    return config{
        Addr:        os.Getenv("VAULT_ADDR"),
        Namespace:   os.Getenv("VAULT_NAMESPACE"),
        Role:        os.Getenv("VAULT_ROLE"),
        AuthMethod:  os.Getenv("VAULT_AUTH_METHOD"),
        SecretPaths: paths,
        Insecure:    insecure,
    }
}

// minimal Vault client wrapper
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
    if v.namespace != "" {
        req.Header.Set("X-Vault-Namespace", v.namespace)
    }
    resp, err := v.httpc.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    var out struct{ Auth struct{ ClientToken string `json:"client_token"` } `json:"auth"` }
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return "", err
    }
    return out.Auth.ClientToken, nil
}

// fetchKVv2Paths fetches secrets from provided kv v2 paths and returns merged map
func (v *vaultClient) fetchKVv2Paths(paths []string) (map[string]string, error) {
    res := map[string]string{}
    for _, p := range paths {
        url := strings.TrimRight(v.addr, "/") + "/v1/" + strings.TrimPrefix(p, "/")
        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Set("X-Vault-Token", v.token)
        if v.namespace != "" {
            req.Header.Set("X-Vault-Namespace", v.namespace)
        }
        resp, err := v.httpc.Do(req)
        if err != nil {
            return nil, err
        }
        var body map[string]interface{}
        if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
            resp.Body.Close()
            return nil, err
        }
        resp.Body.Close()
        // KV v2 returns {data:{data: {...}}}
        if d, ok := body["data"].(map[string]interface{}); ok {
            if inner, ok := d["data"].(map[string]interface{}); ok {
                for k, v := range inner {
                    res[k] = fmt.Sprintf("%v", v)
                }
            }
        }
    }
    return res, nil
}

// minimal insecure TLS config placeholder
var tlsConfigInsecure = tls.Config{InsecureSkipVerify: true}
