package portainer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListStacksFiltersClientSide(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stacks", func(w http.ResponseWriter, r *http.Request) {
		// Verify no filter query params are sent
		if q := r.URL.RawQuery; q != "" {
			t.Errorf("expected no query params, got %q", q)
		}
		stacks := []Stack{
			{ID: 1, Name: "stack-ep1", EndpointID: 1, Status: 1},
			{ID: 2, Name: "stack-ep2", EndpointID: 2, Status: 1},
			{ID: 3, Name: "stack-ep1-b", EndpointID: 1, Status: 1},
			{ID: 4, Name: "stack-ep3", EndpointID: 3, Status: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stacks)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")

	stacks, err := client.ListStacks(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListStacks: %v", err)
	}
	if len(stacks) != 2 {
		t.Fatalf("got %d stacks, want 2", len(stacks))
	}
	for _, s := range stacks {
		if s.EndpointID != 1 {
			t.Errorf("stack %q has EndpointID %d, want 1", s.Name, s.EndpointID)
		}
	}
	if stacks[0].Name != "stack-ep1" {
		t.Errorf("first stack: got %q, want %q", stacks[0].Name, "stack-ep1")
	}
	if stacks[1].Name != "stack-ep1-b" {
		t.Errorf("second stack: got %q, want %q", stacks[1].Name, "stack-ep1-b")
	}
}

func TestListStacksNoFilter(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stacks", func(w http.ResponseWriter, r *http.Request) {
		stacks := []Stack{
			{ID: 1, Name: "stack-ep1", EndpointID: 1, Status: 1},
			{ID: 2, Name: "stack-ep2", EndpointID: 2, Status: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stacks)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")

	stacks, err := client.ListStacks(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListStacks: %v", err)
	}
	if len(stacks) != 2 {
		t.Fatalf("got %d stacks, want 2", len(stacks))
	}
}

func TestListStacksNoMatchReturnsEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stacks", func(w http.ResponseWriter, r *http.Request) {
		stacks := []Stack{
			{ID: 1, Name: "stack-ep2", EndpointID: 2, Status: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stacks)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")

	stacks, err := client.ListStacks(context.Background(), 99)
	if err != nil {
		t.Fatalf("ListStacks: %v", err)
	}
	if len(stacks) != 0 {
		t.Fatalf("got %d stacks, want 0", len(stacks))
	}
}
