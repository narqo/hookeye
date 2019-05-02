package hooks

import (
	"context"
	"encoding/json"
	"log"

	"github.com/adjust/hookeye/github"
	"github.com/adjust/hookeye/hooks/githubsvc"
	"github.com/adjust/hookeye/stream"
	"golang.org/x/xerrors"
)

// repo -> resourcePath
var repoToProject = map[string]string{
	"backend": "/orgs/adjust/projects/13", // "/adjust/backend/projects/1" for repo level project
}

type IssuesProcessor struct {
	GithubService *githubsvc.Service
}

func (p *IssuesProcessor) Process(ctx context.Context, msg *stream.Message) error {
	var issue github.Issue
	if err := json.Unmarshal(msg.Data, &issue); err != nil {
		return xerrors.Errorf("failed to unmarshal message %d: %w", msg.Offset, err)
	}

	cards, err := p.GithubService.IssueProjectCards(ctx, issue.NodeID)
	if err != nil {
		return xerrors.Errorf("failed to get issue project cards: %w", err)
	}

	repo := cards.Node.Repository
	projPath := repoToProject[repo.Name]
	if projPath == "" {
		log.Printf("no project for repo %q\n", repo)
		return nil
	}

	nodes := cards.Node.ProjectCards.Nodes
	if len(nodes) == 0 {
		return p.createIssueProjectCard(ctx, projPath, issue.NodeID)
	}

	for _, node := range nodes {
		if node.Project.ResourcePath == projPath {
			log.Printf("nothing to be done for repo %v, issue %v\n", repo, issue)
			return nil
		}
	}

	return p.createIssueProjectCard(ctx, projPath, issue.NodeID)
}

func (p *IssuesProcessor) createIssueProjectCard(ctx context.Context, projPath, issueID string) error {
	projID, err := p.GithubService.FindProjectID(ctx, projPath)
	if err != nil {
		return xerrors.Errorf("failed to get project id for %q: %w", projPath, err)
	}

	_, err = p.GithubService.AddIssueProjectCard(ctx, issueID, string(projID.ID))
	if err != nil {
		return xerrors.Errorf("failed to add project card to issue %s, project %s: %w", issueID, projID.ID, err)
	}

	return nil
}
