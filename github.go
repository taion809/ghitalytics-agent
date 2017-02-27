package main

import (
	"context"
	"time"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

type GitApi interface {
	ListRepositories(context.Context) ([]string, error)
}

type GithubApi struct {
	organization string
	client       *github.Client
}

func NewDefaultGithubApi(ctx context.Context, orgName, accessToken string) *GithubApi {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)

	tc := oauth2.NewClient(ctx, ts)
	return &GithubApi{organization: orgName, client: github.NewClient(tc)}
}

func (g *GithubApi) ListRepositories(ctx context.Context) ([]string, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	repos := []string{}
	opt := &github.RepositoryListByOrgOptions{
		Type: "sources",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		r, resp, err := g.client.Repositories.ListByOrg(timeout, g.organization, opt)
		if err != nil {
			return repos, err
		}

		for _, v := range r {
			repos = append(repos, v.GetName())
		}

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	return repos, nil
}
