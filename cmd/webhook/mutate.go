package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    admissionv1 "k8s.io/api/admission/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
    annEnabled     = "vault.example.com/enabled"
    annAuthMethod  = "vault.example.com/auth-method"
    annRole        = "vault.example.com/role"
    annAddr        = "vault.example.com/addr"
    annNamespace   = "vault.example.com/namespace"
    annSecretPaths = "vault.example.com/secret-paths"
)

func mutateHandler(w http.ResponseWriter, r *http.Request) {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "could not read request", http.StatusBadRequest)
        return
    }

    var review admissionv1.AdmissionReview
    if err := json.Unmarshal(body, &review); err != nil {
        http.Error(w, "invalid admission review", http.StatusBadRequest)
        return
    }

    resp := admissionv1.AdmissionResponse{Allowed: true}

    req := review.Request
    if req == nil {
        review.Response = &resp
        writeResponse(w, &review)
        return
    }

    if req.Kind.Kind != "Pod" || req.Operation != admissionv1.Create {
        review.Response = &resp
        writeResponse(w, &review)
        return
    }

    var pod corev1.Pod
    if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
        resp.Result = metaStatus("failed to decode pod")
        review.Response = &resp
        writeResponse(w, &review)
        return
    }

    ann := pod.ObjectMeta.Annotations
    if ann == nil || ann[annEnabled] != "true" {
        review.Response = &resp
        writeResponse(w, &review)
        return
    }

    patches := generatePatches(&pod, ann)
    if len(patches) > 0 {
        bs, _ := json.Marshal(patches)
        pt := admissionv1.PatchTypeJSONPatch
        resp.Patch = bs
        resp.PatchType = &pt
    }

    review.Response = &resp
    writeResponse(w, &review)
}

func writeResponse(w http.ResponseWriter, review *admissionv1.AdmissionReview) {
    review.Kind = "AdmissionReview"
    review.APIVersion = "admission.k8s.io/v1"
    out, _ := json.Marshal(review)
    w.Header().Set("Content-Type", "application/json")
    w.Write(out)
}

func metaStatus(msg string) *metav1.Status {
    return &metav1.Status{Message: msg}
}

// generatePatches creates JSONPatch ops according to the request
func generatePatches(pod *corev1.Pod, ann map[string]string) []map[string]interface{} {
    patches := []map[string]interface{}{}

    // add shared emptyDir volume
    vol := map[string]interface{}{
        "name": "vault-env-shared",
        "emptyDir": map[string]interface{}{},
    }
    patches = append(patches, map[string]interface{}{"op": "add", "path": "/spec/volumes/-", "value": vol})

    // add init container that copies binary into shared volume
    image := ann["vault.example.com/init-image"]
    if image == "" {
        image = "example/vault-env-runner:latest"
    }
    initc := map[string]interface{}{
        "name":  "vault-env-init",
        "image": image,
        "command": []string{"/bin/sh", "-c", "cp /usr/local/bin/vault-env-runner /vault-env/ && chmod +x /vault-env/vault-env-runner"},
        "volumeMounts": []map[string]interface{}{{"name": "vault-env-shared", "mountPath": "/vault-env"}},
    }
    patches = append(patches, map[string]interface{}{"op": "add", "path": "/spec/initContainers/-", "value": initc})

    // for each app container, mount shared volume, rewrite command, and inject envs
    for i, c := range pod.Spec.Containers {
        // add volumeMount
        vm := map[string]interface{}{"name": "vault-env-shared", "mountPath": "/vault-env"}
        if len(c.VolumeMounts) == 0 {
            // add env array if needed
            patches = append(patches, map[string]interface{}{"op": "add", "path": fmtPath("/spec/containers/%d/volumeMounts", i), "value": []map[string]interface{}{vm}})
        } else {
            patches = append(patches, map[string]interface{}{"op": "add", "path": fmtPath("/spec/containers/%d/volumeMounts/-", i), "value": vm})
        }

        // preserve original command and args as JSON strings in env
        var origCmd []string
        if len(c.Command) > 0 {
            origCmd = c.Command
        }
        var origArgs []string
        if len(c.Args) > 0 {
            origArgs = c.Args
        }
        cmdVal, _ := json.Marshal(origCmd)
        argsVal, _ := json.Marshal(origArgs)

        // replace command to run vault-env-runner
        patches = append(patches, map[string]interface{}{"op": "replace", "path": fmtPath("/spec/containers/%d/command", i), "value": []string{"/vault-env/vault-env-runner"}})

        // ensure env array exists
        if len(c.Env) == 0 {
            patches = append(patches, map[string]interface{}{"op": "add", "path": fmtPath("/spec/containers/%d/env", i), "value": []map[string]interface{}{} })
        }

        // add ORIGINAL command and args env vars
        patches = append(patches, map[string]interface{}{"op": "add", "path": fmtPath("/spec/containers/%d/env/-", i), "value": map[string]interface{}{"name": "VAULT_ORIG_CMD", "value": string(cmdVal)}})
        patches = append(patches, map[string]interface{}{"op": "add", "path": fmtPath("/spec/containers/%d/env/-", i), "value": map[string]interface{}{"name": "VAULT_ORIG_ARGS", "value": string(argsVal)}})

        // inject vault config envs
        envs := []map[string]string{
            {"name": "VAULT_ADDR", "value": ann[annAddr]},
            {"name": "VAULT_NAMESPACE", "value": ann[annNamespace]},
            {"name": "VAULT_ROLE", "value": ann[annRole]},
            {"name": "VAULT_AUTH_METHOD", "value": ann[annAuthMethod]},
            {"name": "VAULT_SECRET_PATHS", "value": ann[annSecretPaths]},
        }
        for _, e := range envs {
            if e["value"] == "" {
                continue
            }
            patches = append(patches, map[string]interface{}{"op": "add", "path": fmtPath("/spec/containers/%d/env/-", i), "value": map[string]interface{}{"name": e["name"], "value": e["value"]}})
        }
    }

    return patches
}

func fmtPath(pattern string, i int) string {
    return fmt.Sprintf(pattern, i)
}
