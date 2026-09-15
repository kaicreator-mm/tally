package gitengineering

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrProviderRequest = errors.New("gitengineering: provider request failed")

// GitHubAdapter is the first RemoteProvider implementation. Provider-specific
// REST fields stay inside the adapter so GitHub never becomes the domain model.
type GitHubAdapter struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func NewGitHubAdapter(token string, client *http.Client) *GitHubAdapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &GitHubAdapter{BaseURL: "https://api.github.com", Token: token, Client: client}
}

func (g *GitHubAdapter) endpoint(repo, suffix string) (string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("gitengineering: repository must be owner/name")
	}
	base := strings.TrimRight(g.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	return base + "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + suffix, nil
}

func (g *GitHubAdapter) do(ctx context.Context, method, endpoint string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(g.Token) != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderRequest, err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%w: read response: %v", ErrProviderRequest, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(payload, &apiErr)
		msg := strings.TrimSpace(apiErr.Message)
		if msg == "" {
			msg = strings.TrimSpace(string(payload))
		}
		return fmt.Errorf("%w: github status %d: %s", ErrProviderRequest, resp.StatusCode, msg)
	}
	if out != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("%w: decode response: %v", ErrProviderRequest, err)
		}
	}
	return nil
}

func (g *GitHubAdapter) CreateIssue(ctx context.Context, repo string, in IssueInput) (RemoteArtifact, error) {
	ep, err := g.endpoint(repo, "/issues")
	if err != nil {
		return RemoteArtifact{}, err
	}
	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := g.do(ctx, http.MethodPost, ep, map[string]any{"title": in.Title, "body": in.Body}, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: fmt.Sprint(out.Number), URI: out.HTMLURL}, nil
}

func (g *GitHubAdapter) UpdateIssue(ctx context.Context, repo, issueID string, in IssueInput) (RemoteArtifact, error) {
	ep, err := g.endpoint(repo, "/issues/"+url.PathEscape(issueID))
	if err != nil {
		return RemoteArtifact{}, err
	}
	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := g.do(ctx, http.MethodPatch, ep, map[string]any{"title": in.Title, "body": in.Body}, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: fmt.Sprint(out.Number), URI: out.HTMLURL}, nil
}

func (g *GitHubAdapter) CloseIssue(ctx context.Context, repo, issueID string) (RemoteArtifact, error) {
	ep, err := g.endpoint(repo, "/issues/"+url.PathEscape(issueID))
	if err != nil {
		return RemoteArtifact{}, err
	}
	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := g.do(ctx, http.MethodPatch, ep, map[string]any{"state": "closed"}, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: fmt.Sprint(out.Number), URI: out.HTMLURL}, nil
}

func (g *GitHubAdapter) OpenPullRequest(ctx context.Context, repo string, in PullRequestInput) (RemoteArtifact, error) {
	ep, err := g.endpoint(repo, "/pulls")
	if err != nil {
		return RemoteArtifact{}, err
	}
	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		Head    struct {
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
	}
	body := map[string]any{"title": in.Title, "body": in.Body, "head": in.Head, "base": in.Base}
	if err := g.do(ctx, http.MethodPost, ep, body, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: fmt.Sprint(out.Number), URI: out.HTMLURL, Revision: out.Head.SHA, Base: out.Base.Ref}, nil
}

func (g *GitHubAdapter) UpdatePullRequest(ctx context.Context, repo, pullRequestID string, in PullRequestInput) (RemoteArtifact, error) {
	ep, err := g.endpoint(repo, "/pulls/"+url.PathEscape(pullRequestID))
	if err != nil {
		return RemoteArtifact{}, err
	}
	body := map[string]any{}
	if in.Title != "" {
		body["title"] = in.Title
	}
	if in.Body != "" {
		body["body"] = in.Body
	}
	if in.Base != "" {
		body["base"] = in.Base
	}
	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		Head    struct {
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}
	if err := g.do(ctx, http.MethodPatch, ep, body, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: fmt.Sprint(out.Number), URI: out.HTMLURL, Revision: out.Head.SHA, Base: out.Base.Ref}, nil
}

func (g *GitHubAdapter) ReviewPullRequest(ctx context.Context, repo, pullRequestID string, in ReviewInput) (RemoteArtifact, error) {
	ep, err := g.endpoint(repo, "/pulls/"+url.PathEscape(pullRequestID)+"/reviews")
	if err != nil {
		return RemoteArtifact{}, err
	}
	event := "COMMENT"
	switch in.State {
	case ReviewApprove:
		event = "APPROVE"
	case ReviewRequestChanges:
		event = "REQUEST_CHANGES"
	case ReviewComment:
	default:
		return RemoteArtifact{}, fmt.Errorf("gitengineering: invalid review state %q", in.State)
	}
	body := map[string]any{"event": event, "body": in.Body}
	if in.HeadRevision != "" {
		body["commit_id"] = in.HeadRevision
	}
	var out struct {
		ID       int64  `json:"id"`
		HTMLURL  string `json:"html_url"`
		CommitID string `json:"commit_id"`
	}
	if err := g.do(ctx, http.MethodPost, ep, body, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: fmt.Sprint(out.ID), URI: out.HTMLURL, Revision: out.CommitID}, nil
}

func (g *GitHubAdapter) MergePullRequest(ctx context.Context, repo, pullRequestID string, in MergeInput) (RemoteArtifact, error) {
	ep, err := g.endpoint(repo, "/pulls/"+url.PathEscape(pullRequestID)+"/merge")
	if err != nil {
		return RemoteArtifact{}, err
	}
	body := map[string]any{"sha": in.ExpectedHeadRevision}
	if in.Method != "" {
		body["merge_method"] = in.Method
	}
	var out struct {
		SHA     string `json:"sha"`
		Merged  bool   `json:"merged"`
		Message string `json:"message"`
	}
	if err := g.do(ctx, http.MethodPut, ep, body, &out); err != nil {
		return RemoteArtifact{}, err
	}
	if !out.Merged || strings.TrimSpace(out.SHA) == "" {
		return RemoteArtifact{}, fmt.Errorf("%w: merge not completed: %s", ErrProviderRequest, out.Message)
	}
	return RemoteArtifact{ExternalID: pullRequestID, Revision: out.SHA}, nil
}

func (g *GitHubAdapter) CreateTag(ctx context.Context, repo string, in TagInput) (RemoteArtifact, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Revision) == "" {
		return RemoteArtifact{}, errors.New("gitengineering: tag name and revision are required")
	}
	ep, err := g.endpoint(repo, "/git/refs")
	if err != nil {
		return RemoteArtifact{}, err
	}
	var out struct {
		Ref    string `json:"ref"`
		URL    string `json:"url"`
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := g.do(ctx, http.MethodPost, ep, map[string]any{"ref": "refs/tags/" + in.Name, "sha": in.Revision}, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: out.Ref, URI: out.URL, Revision: out.Object.SHA}, nil
}

func (g *GitHubAdapter) PublishRelease(ctx context.Context, repo string, in ReleaseInput) (RemoteArtifact, error) {
	if strings.TrimSpace(in.TagName) == "" || strings.TrimSpace(in.Revision) == "" {
		return RemoteArtifact{}, errors.New("gitengineering: release tag and revision are required")
	}
	ep, err := g.endpoint(repo, "/releases")
	if err != nil {
		return RemoteArtifact{}, err
	}
	body := map[string]any{"tag_name": in.TagName, "name": in.Name, "body": in.Body, "target_commitish": in.Revision, "draft": false, "prerelease": false}
	var out struct {
		ID              int64  `json:"id"`
		HTMLURL         string `json:"html_url"`
		TagName         string `json:"tag_name"`
		TargetCommitish string `json:"target_commitish"`
	}
	if err := g.do(ctx, http.MethodPost, ep, body, &out); err != nil {
		return RemoteArtifact{}, err
	}
	return RemoteArtifact{ExternalID: fmt.Sprint(out.ID), URI: out.HTMLURL, Revision: in.Revision}, nil
}
