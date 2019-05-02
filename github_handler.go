package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/adjust/hookeye/stream"
	"golang.org/x/xerrors"
)

const (
	GithubEventIssues = "issues"
)

type EventAction string

const (
	EventOpened     = EventAction("opened")
	EventEdited     = EventAction("edited")
	EventDeleted    = EventAction("deleted")
	EventClosed     = EventAction("closed")
	EventReopened   = EventAction("reopened")
	EventAssigned   = EventAction("assigned")
	EventUnassigned = EventAction("unassigned")
)

var ErrNoSignature = xerrors.New("no signature")

type IssuesEventPayload struct {
	Action EventAction     `json:"action"`
	Issue  json.RawMessage `json:"issue"`
}

type GithubHandler struct {
	stream *stream.Stream
	secret string
}

func NewGithubHandler(stream *stream.Stream, secret string) *GithubHandler {
	return &GithubHandler{stream, secret}
}

func (h *GithubHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/github", h)
}

func (h *GithubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		HandleErrorHTTP(
			StatusError(http.StatusMethodNotAllowed, "method not allowed", nil), w, r)
		return
	} else if event := r.Header.Get("X-GitHub-Event"); event != GithubEventIssues {
		HandleErrorHTTP(
			StatusError(http.StatusBadRequest, fmt.Sprintf("not supported event %q", event), nil), w, r)
		return
	}

	err := h.handleIssuesRequest(w, r)
	if err != nil {
		HandleErrorHTTP(err, w, r)
		return
	}

	io.WriteString(w, "OK")
}

func (h *GithubHandler) handleIssuesRequest(w http.ResponseWriter, r *http.Request) error {
	event := &IssuesEventPayload{}
	if err := readRequest(r, h.secret, event); err != nil {
		return StatusError(http.StatusBadRequest, "bad event", err)
	}

	switch event.Action {
	case EventOpened:
		return h.handleIssueOpened(r.Context(), event)
	default:
		return StatusError(http.StatusBadRequest, fmt.Sprintf("not supported event action %q", event.Action), nil)
	}
}

func (h *GithubHandler) handleIssueOpened(ctx context.Context, event *IssuesEventPayload) error {
	return h.stream.Push(ctx, githubIssuesTopic, event.Issue)
}

func readRequest(r *http.Request, secret string, v interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	hubSig := r.Header.Get("X-Hub-Signature")
	if err := verifyRequest(secret, hubSig, body); err != nil {
		return err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return xerrors.Errorf("could not decode event payload %s: %w", body, err)
	}
	return nil
}

func verifyRequest(secret, sig string, body []byte) error {
	if secret != "" && sig == "" {
		return ErrNoSignature
	}

	hash := hmac.New(sha1.New, []byte(secret))
	hash.Write(body)

	sig1 := hash.Sum([]byte(nil))
	sig2, err := hex.DecodeString(strings.TrimPrefix(sig, "sha1="))
	if err != nil {
		return err
	}
	if !hmac.Equal(sig1, sig2) {
		return xerrors.Errorf("bad signature %s", sig)
	}
	return nil
}
