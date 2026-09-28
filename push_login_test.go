package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPollLoginPush_States(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string
		want   string
		token  string
	}{
		"pending":  {http.StatusAccepted, `{"status":"pending"}`, PushPending, ""},
		"denied":   {http.StatusForbidden, `{"status":"denied"}`, PushDenied, ""},
		"expired":  {http.StatusGone, `{"status":"expired"}`, PushExpired, ""},
		"approved": {http.StatusOK, `{"jwt":"session.jwt","active_workspace":"ws"}`, PushApproved, "session.jwt"},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/auth/accounts/login/mfa/push/c1" {
					t.Errorf("path = %s", r.URL.Path)
				}
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["mfa_token"] != "mfa" || body["poll_token"] != "poll" {
					t.Errorf("body = %v", body)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL))
			resp, state, err := client.PollLoginPush(context.Background(), &PushChallenge{ChallengeID: "c1", PollToken: "poll"}, "mfa")
			if err != nil {
				t.Fatal(err)
			}
			if state != tc.want {
				t.Fatalf("state = %s, want %s", state, tc.want)
			}
			if tc.token != "" && (resp == nil || resp.Token != tc.token) {
				t.Fatalf("resp = %+v", resp)
			}
			if tc.token == "" && resp != nil {
				t.Fatalf("unexpected login for %s", name)
			}
		})
	}
}
