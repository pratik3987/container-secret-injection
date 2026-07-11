package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSkipWhenNotEnabled(t *testing.T) {
	pod := corev1.Pod{}
	req := admissionv1.AdmissionReview{}
	req.Request = &admissionv1.AdmissionRequest{Kind: metav1.GroupVersionKind{Kind: "Pod"}, Operation: admissionv1.Create}
	raw, _ := json.Marshal(pod)
	req.Request.Object.Raw = raw

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/mutate", bytesReader(body))
	w := httptest.NewRecorder()
	mutateHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if resp.Response == nil || resp.Response.Patch != nil {
		t.Fatalf("expected no patch, got %v", resp.Response)
	}
}

func TestPatchGeneratedWhenEnabled(t *testing.T) {
	pod := corev1.Pod{}
	pod.ObjectMeta.Annotations = map[string]string{"vault.prtk.com/inject": "true", "vault.prtk.com/vault-addr": "https://vault.prtk.com", "vault.prtk.com/vault-path": "kv/data/app"}
	req := admissionv1.AdmissionReview{}
	req.Request = &admissionv1.AdmissionRequest{Kind: metav1.GroupVersionKind{Kind: "Pod"}, Operation: admissionv1.Create}
	raw, _ := json.Marshal(pod)
	req.Request.Object.Raw = raw

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/mutate", bytesReader(body))
	w := httptest.NewRecorder()
	mutateHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if resp.Response == nil || resp.Response.Patch == nil {
		t.Fatalf("expected patch, got %v", resp.Response)
	}
}

// helper to create io.Reader from bytes
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
