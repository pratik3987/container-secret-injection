package main

import (
    "flag"
    "fmt"
    "log"
    "net/http"
)

func main() {
    var addr string
    var certFile string
    var keyFile string
    flag.StringVar(&addr, "listen", ":8443", "address to listen on")
    flag.StringVar(&certFile, "tls-cert-file", "/tls/tls.crt", "TLS cert file")
    flag.StringVar(&keyFile, "tls-key-file", "/tls/tls.key", "TLS key file")
    flag.Parse()

    mux := http.NewServeMux()
    mux.HandleFunc("/mutate", mutateHandler)

    srv := &http.Server{Addr: addr, Handler: mux}
    log.Printf("starting webhook server on %s", addr)
    err := srv.ListenAndServeTLS(certFile, keyFile)
    if err != nil {
        log.Fatalf("server failed: %v", err)
    }
    fmt.Println("exiting")
}
