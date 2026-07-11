package controllers

import (
	"context"
	"fmt"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/example/vault-webxy/api/v1alpha1"
)

type VaultInjectorReconciler struct {
	client.Client
}

func (r *VaultInjectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var inst v1alpha1.VaultInjector
	if err := r.Get(ctx, req.NamespacedName, &inst); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Read CA secret
	var s corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: inst.Spec.ServiceNamespace, Name: inst.Spec.CASecret}, &s); err != nil {
		return ctrl.Result{}, fmt.Errorf("get secret: %w", err)
	}
	ca, ok := s.Data["ca.crt"]
	if !ok {
		return ctrl.Result{}, fmt.Errorf("secret missing ca.crt")
	}

	mwc := &admissionv1.MutatingWebhookConfiguration{}
	mwc.Name = "vault-webhook"
	mwc.Webhooks = []admissionv1.MutatingWebhook{{
		Name: "vault.prtk.com",
		ClientConfig: admissionv1.WebhookClientConfig{
			Service:  &admissionv1.ServiceReference{Name: inst.Spec.ServiceName, Namespace: inst.Spec.ServiceNamespace, Path: ptrString("/mutate")},
			CABundle: []byte(ca),
		},
		Rules: []admissionv1.RuleWithOperations{{
			Operations: []admissionv1.OperationType{admissionv1.Create},
			Rule:       admissionv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"pods"}},
		}},
		AdmissionReviewVersions: []string{"v1"},
		SideEffects:             ptrSideEffect(admissionv1.SideEffectClassNone),
	}}

	// create or update
	_ = r.Delete(ctx, mwc)
	if err := r.Create(ctx, mwc); err != nil {
		return ctrl.Result{}, fmt.Errorf("create mwc: %w", err)
	}

	return ctrl.Result{}, nil
}

func ptrString(s string) *string                                               { return &s }
func ptrSideEffect(s admissionv1.SideEffectClass) *admissionv1.SideEffectClass { return &s }

func (r *VaultInjectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VaultInjector{}).
		Complete(r)
}
