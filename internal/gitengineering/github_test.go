package gitengineering

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestGitHubAdapterIssuePRMergeTagRelease(t *testing.T) {
	var mu sync.Mutex
	seen := []string{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/o/r/issues":
			json.NewEncoder(w).Encode(map[string]any{"number": 36, "html_url": "https://github.test/o/r/issues/36"})
		case "/repos/o/r/pulls":
			json.NewEncoder(w).Encode(map[string]any{"number": 40, "html_url": "https://github.test/o/r/pull/40", "head": map[string]any{"sha": "head"}, "base": map[string]any{"ref": "main", "sha": "base"}})
		case "/repos/o/r/pulls/40/merge":
			json.NewEncoder(w).Encode(map[string]any{"sha": "merge-sha", "merged": true, "message": "ok"})
		case "/repos/o/r/git/refs":
			json.NewEncoder(w).Encode(map[string]any{"ref": "refs/tags/v1.5.3", "url": "https://api.test/ref", "object": map[string]any{"sha": "merge-sha"}})
		case "/repos/o/r/releases":
			json.NewEncoder(w).Encode(map[string]any{"id": 9, "html_url": "https://github.test/o/r/releases/9", "tag_name": "v1.5.3", "target_commitish": "merge-sha"})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer s.Close()
	g := NewGitHubAdapter("token", s.Client())
	g.BaseURL = s.URL
	ctx := context.Background()
	issue, err := g.CreateIssue(ctx, "o/r", IssueInput{Title: "x"})
	if err != nil || issue.ExternalID != "36" {
		t.Fatalf("issue=%+v err=%v", issue, err)
	}
	pr, err := g.OpenPullRequest(ctx, "o/r", PullRequestInput{Title: "p", Head: "feat", Base: "main"})
	if err != nil || pr.ExternalID != "40" || pr.Revision != "head" {
		t.Fatalf("pr=%+v err=%v", pr, err)
	}
	m, err := g.MergePullRequest(ctx, "o/r", "40", MergeInput{ExpectedHeadRevision: "head", Method: "squash"})
	if err != nil || m.Revision != "merge-sha" {
		t.Fatalf("merge=%+v err=%v", m, err)
	}
	tag, err := g.CreateTag(ctx, "o/r", TagInput{Name: "v1.5.3", Revision: "merge-sha"})
	if err != nil || tag.Revision != "merge-sha" {
		t.Fatalf("tag=%+v err=%v", tag, err)
	}
	rel, err := g.PublishRelease(ctx, "o/r", ReleaseInput{TagName: "v1.5.3", Name: "v1.5.3", Revision: "merge-sha"})
	if err != nil || rel.Revision != "merge-sha" {
		t.Fatalf("release=%+v err=%v", rel, err)
	}
	if len(seen) != 5 {
		t.Fatalf("seen=%v", seen)
	}
}

func TestGitHubAdapterMapsProviderFailure(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"stale head"}`))
	}))
	defer s.Close()
	g := NewGitHubAdapter("", s.Client())
	g.BaseURL = s.URL
	_, err := g.MergePullRequest(context.Background(), "o/r", "1", MergeInput{ExpectedHeadRevision: "old"})
	if !errors.Is(err, ErrProviderRequest) {
		t.Fatalf("got %v", err)
	}
}
