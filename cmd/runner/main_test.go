package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
)

func TestLoginAndFetchKVv2(t *testing.T) {
    // mock Vault server
    mux := http.NewServeMux()
    mux.HandleFunc("/v1/auth/kubernetes/login", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]interface{}{"auth": map[string]interface{}{"client_token": "s.test"}})
    })
    mux.HandleFunc("/v1/secret/data/app", func(w http.ResponseWriter, r *http.Request) {
        _ = json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{"data": map[string]interface{}{"DB_PASS": "hunter2"}}})
    })
    s := httptest.NewServer(mux)
    defer s.Close()

    cfg := config{Addr: s.URL, Role: "demo-role", Namespace: ""}
    client := newVaultClient(cfg)
    token, err := client.loginWithKubernetes()
    if err != nil {
        t.Fatalf("login failed: %v", err)
    }
    client.token = token
    secrets, err := client.fetchKVv2Paths([]string{"secret/data/app"})
    if err != nil {
        t.Fatalf("fetch failed: %v", err)
    }
    if secrets["DB_PASS"] != "hunter2" {
        t.Fatalf("unexpected secret value: %v", secrets)
    }
    // ensure env export doesn't crash
    os.Setenv("VAULT_ORIG_CMD", "[]")
    os.Setenv("VAULT_ORIG_ARGS", "[]")
}
