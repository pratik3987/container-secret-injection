package main

import (
    "flag"
    "os"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client/config"

    apiv1 "github.com/example/vault-webxy/api/v1alpha1"
    "github.com/example/vault-webxy/controllers"
)

func main() {
    mgrAddr := flag.String("metrics-addr", ":8080", "metrics address")
    flag.Parse()

    cfg := config.GetConfigOrDie()
    mgr, err := ctrl.NewManager(cfg, ctrl.Options{MetricsBindAddress: *mgrAddr})
    if err != nil {
        os.Exit(1)
    }
    if err := apiv1.AddToScheme(mgr.GetScheme()); err != nil {
        os.Exit(1)
    }
    if err := (&controllers.VaultInjectorReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
        os.Exit(1)
    }
    if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
        os.Exit(1)
    }
}
