package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const postgresBase = "/v1/workspaces/ws/projects/project/envs/prod/svcs/db/postgres"

func jsonResponse(r *http.Request, body string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func testServices(transport http.RoundTripper) *ServicesClient {
	client := NewClient(
		WithBaseURL("https://api.example.test"),
		WithHTTPClient(&http.Client{Transport: transport}),
	)
	return client.Workspace("ws").Project("project").Env("prod").Services()
}

func TestCreatePostgresDatabase(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != postgresBase+"/databases" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var input CreatePostgresDatabaseInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.Name != "analytics" || input.Owner != "reporter" {
			t.Fatalf("unexpected input: %#v", input)
		}
		return jsonResponse(r, `{"name":"analytics","owner":"reporter"}`)
	})

	got, err := testServices(transport).CreatePostgresDatabase(context.Background(), "db",
		CreatePostgresDatabaseInput{Name: "analytics", Owner: "reporter"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "analytics" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

// A database name is a user-supplied identifier that lands in the path, so it
// must be escaped rather than concatenated.
func TestDeletePostgresDatabaseEscapesName(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.EscapedPath() != postgresBase+"/databases/we%2Frd" {
			t.Fatalf("unescaped path: %s", r.URL.EscapedPath())
		}
		return jsonResponse(r, `{}`)
	})

	if err := testServices(transport).DeletePostgresDatabase(context.Background(), "db", "we/rd"); err != nil {
		t.Fatal(err)
	}
}

// Installed must hide the "absent" tombstones the platform keeps so it can finish
// dropping an extension — reporting one as installed would be wrong.
func TestPostgresExtensionsInstalled(t *testing.T) {
	extensions := PostgresExtensions{Extensions: []PostgresExtension{
		{Name: "pg_trgm", Ensure: PostgresExtensionPresent},
		{Name: "hstore", Ensure: PostgresExtensionAbsent},
		{Name: "unaccent", Ensure: PostgresExtensionPresent},
	}}

	got := extensions.Installed()
	if len(got) != 2 || got[0] != "pg_trgm" || got[1] != "unaccent" {
		t.Fatalf("got %v, want only the present extensions", got)
	}
}

func TestEnablePostgresExtensionPreservesExisting(t *testing.T) {
	var sent SetPostgresExtensionsInput
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		path := postgresBase + "/databases/app/extensions"
		if r.URL.Path != path {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			return jsonResponse(r, `{"extensions":[{"name":"hstore","ensure":"present"},`+
				`{"name":"citext","ensure":"absent"}],"supported":["hstore","pg_trgm","citext"]}`)
		case http.MethodPut:
			if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
				t.Fatal(err)
			}
			return jsonResponse(r, `{"extensions":[{"name":"hstore","ensure":"present"},`+
				`{"name":"pg_trgm","ensure":"present"}]}`)
		default:
			t.Fatalf("unexpected method: %s", r.Method)
			return nil, nil
		}
	})

	got, err := testServices(transport).EnablePostgresExtension(context.Background(), "db", "app", "pg_trgm")
	if err != nil {
		t.Fatal(err)
	}
	// The already-installed extension must survive, and the absent tombstone must
	// not be resurrected by being echoed back as desired.
	if len(sent.Extensions) != 2 || sent.Extensions[0] != "hstore" || sent.Extensions[1] != "pg_trgm" {
		t.Fatalf("sent %v, want the existing set plus the new extension", sent.Extensions)
	}
	if len(got.Installed()) != 2 {
		t.Fatalf("unexpected response: %#v", got)
	}
}

// Enabling something already installed must not issue a write at all.
func TestEnablePostgresExtensionIsIdempotent(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected write: %s %s", r.Method, r.URL.Path)
		}
		return jsonResponse(r, `{"extensions":[{"name":"pg_trgm","ensure":"present"}]}`)
	})

	if _, err := testServices(transport).EnablePostgresExtension(context.Background(), "db", "app", "pg_trgm"); err != nil {
		t.Fatal(err)
	}
}

func TestDisablePostgresExtension(t *testing.T) {
	var sent SetPostgresExtensionsInput
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case http.MethodGet:
			return jsonResponse(r, `{"extensions":[{"name":"hstore","ensure":"present"},`+
				`{"name":"pg_trgm","ensure":"present"}]}`)
		case http.MethodPut:
			if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
				t.Fatal(err)
			}
			return jsonResponse(r, `{"extensions":[{"name":"hstore","ensure":"present"}]}`)
		default:
			t.Fatalf("unexpected method: %s", r.Method)
			return nil, nil
		}
	})

	if _, err := testServices(transport).DisablePostgresExtension(context.Background(), "db", "app", "pg_trgm"); err != nil {
		t.Fatal(err)
	}
	if len(sent.Extensions) != 1 || sent.Extensions[0] != "hstore" {
		t.Fatalf("sent %v, want the remaining extension only", sent.Extensions)
	}
}

// Clearing every extension must send an empty array, not a null the server would
// have to guess at.
func TestSetPostgresExtensionsSendsEmptyArray(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `"extensions":[]`) {
			t.Fatalf("unexpected body: %s", body)
		}
		return jsonResponse(r, `{"extensions":[]}`)
	})

	if _, err := testServices(transport).SetPostgresExtensions(context.Background(), "db", "app",
		SetPostgresExtensionsInput{}); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePostgresUser(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != postgresBase+"/users" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var input CreatePostgresUserInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if !input.Replication || len(input.InRoles) != 1 {
			t.Fatalf("unexpected input: %#v", input)
		}
		return jsonResponse(r, `{"username":"replicator","password":"s3cret","replication":true,`+
			`"in_roles":["pg_read_all_data"]}`)
	})

	got, err := testServices(transport).CreatePostgresUser(context.Background(), "db",
		CreatePostgresUserInput{Username: "replicator", Replication: true, InRoles: []string{"pg_read_all_data"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "s3cret" || !got.Replication {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestUpdatePostgresUserRoles(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPatch || r.URL.Path != postgresBase+"/users/replicator/roles" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		return jsonResponse(r, `{"username":"replicator","in_roles":["pg_create_subscription"]}`)
	})

	got, err := testServices(transport).UpdatePostgresUserRoles(context.Background(), "db", "replicator",
		UpdatePostgresUserRolesInput{InRoles: []string{"pg_create_subscription"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.InRoles) != 1 {
		t.Fatalf("unexpected response: %#v", got)
	}
}
