package githubsvc

import (
	"context"
	"strconv"
	"strings"

	"github.com/adjust/hookeye/github"
	"github.com/machinebox/graphql"
	"golang.org/x/xerrors"
)

const (
	queryIssueProjectCards = `
		query IssueProjectCards ($id: ID!) {
			node(id: $id) {
				... on Issue {
					repository {
						id
						name
						nameWithOwner
					}
					projectCards {
						nodes {
							id
							url
							project {
								id
								name
								url
								resourcePath
							}
						}
					}
				}
			}
		}`

	mutationAddIssueProjectCard = `
		mutation AddIssueProjectCard ($id: ID!, $projectId: ID!) {
			updateIssue(input: {id: $id, projectIds: [$projectId]}) {
				issue {
					repository {
						id
						name
						nameWithOwner
					}
					projectCards {
						nodes {
							id
							url
							project {
								id
								name
								url
								resourcePath
							}
						}
					}
				}
			}
		}`

	queryFindOrdProjectID = `
		query FindProjectID ($login: String!, $number: Int!) {
			organization(login: $login) {
				project(number: $number) {
					id
					name
				}
			}
		}`

	queryFindRepoProjectID = `
		query FindProjectID ($owner: String!, $name: String!, $number: Int!) {
			repository(owner: $owner, name: $name) {
				project(number: $number) {
					id
					name
				}
			}
		}`
)

type Service struct {
	Client *github.Client
}

type IssueProjectCardsResponse struct {
	Node struct {
		Repository   Repository   `json:"repository"`
		ProjectCards ProjectCards `json:"projectCards"`
	} `json:"node"`
}

type Repository struct {
	github.Repository

	NameWithOwner string `json:"nameWithOwner"`
}

type ProjectCards struct {
	Nodes []struct {
		ID      string  `json:"id"`
		Project Project `json:"project"`
	} `json:"nodes"`
}

type Project struct {
	github.Project

	ResourcePath string `json:"resourcePath"`
}

func (svc *Service) IssueProjectCards(ctx context.Context, id string) (*IssueProjectCardsResponse, error) {
	req := graphql.NewRequest(queryIssueProjectCards)
	req.Var("id", id)

	resp := &IssueProjectCardsResponse{}
	err := svc.Client.Run(ctx, req, &resp)
	return resp, err
}

func (svc *Service) AddIssueProjectCard(ctx context.Context, id, projectID string) (*IssueProjectCardsResponse, error) {
	req := graphql.NewRequest(mutationAddIssueProjectCard)
	req.Var("id", id)
	req.Var("projectId", projectID)

	resp := struct {
		UpdateIssue struct {
			IssueProjectCardsResponse
		} `json:"updateIssue"`
	}{}
	if err := svc.Client.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	return &resp.UpdateIssue.IssueProjectCardsResponse, nil
}

type ProjectIDResponse struct {
	github.Project
}

func (svc *Service) FindProjectID(ctx context.Context, projectPath string) (*ProjectIDResponse, error) {
	parts := strings.SplitN(projectPath[1:], "/", 4)
	if len(parts) != 4 {
		return nil, xerrors.Errorf("bad project path %q", projectPath)
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, xerrors.Errorf("could not parse project %q: %w", projectPath, err)
	}
	if parts[0] == "orgs" {
		return svc.findOrgProjectID(ctx, parts[1], number)
	} else {
		return svc.findRepoProjectID(ctx, parts[0], parts[1], number)
	}
}

func (svc *Service) findOrgProjectID(ctx context.Context, login string, number int) (*ProjectIDResponse, error) {
	req := graphql.NewRequest(queryFindOrdProjectID)
	req.Var("login", login)
	req.Var("number", number)

	resp := struct {
		Organization struct {
			Project ProjectIDResponse `json:"project"`
		} `json:"organization"`
	}{}
	if err := svc.Client.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Organization.Project, nil
}

func (svc *Service) findRepoProjectID(ctx context.Context, owner, name string, number int) (*ProjectIDResponse, error) {
	req := graphql.NewRequest(queryFindRepoProjectID)
	req.Var("owner", owner)
	req.Var("name", name)
	req.Var("number", number)

	resp := struct {
		Repository struct {
			Project ProjectIDResponse `json:"project"`
		} `json:"repository"`
	}{}
	if err := svc.Client.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Repository.Project, nil
}
