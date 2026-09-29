package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SetDocuments sends the single-page-app setting only when the caller sets
// it, because the API keeps the current value when it is absent.
func TestStaticSiteSetDocumentsSPAFallback(t *testing.T) {
	const path = "/v1/workspaces/ws/projects/p/envs/prod/svcs/web/site/documents"
	var sent []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		sent = append(sent, body)
		spa, _ := body["spa_fallback"].(bool)
		_ = json.NewEncoder(w).Encode(StaticSite{IndexDocument: "index.html", ErrorDocument: "index.html", SPAFallback: spa})
	}))
	defer server.Close()

	site := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("prod").Services().StaticSite("web")
	ctx := context.Background()

	on := true
	got, err := site.SetDocuments(ctx, UpdateStaticSiteDocumentsInput{SPAFallback: &on})
	if err != nil || !got.SPAFallback {
		t.Fatalf("SetDocuments(on) = %+v, %v", got, err)
	}
	if sent[0]["spa_fallback"] != true {
		t.Errorf("turning on sent %v", sent[0])
	}

	off := false
	if _, err := site.SetDocuments(ctx, UpdateStaticSiteDocumentsInput{SPAFallback: &off}); err != nil {
		t.Fatal(err)
	}
	if v, ok := sent[1]["spa_fallback"]; !ok || v != false {
		t.Errorf("turning off must send false explicitly, sent %v", sent[1])
	}

	if _, err := site.SetDocuments(ctx, UpdateStaticSiteDocumentsInput{IndexDocument: "index.html"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := sent[2]["spa_fallback"]; ok {
		t.Errorf("leaving the setting out must not send it, sent %v", sent[2])
	}
}
