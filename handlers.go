package main

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-github/github"
)

type handler func(context.Context)

func register(ctx context.Context, repo string) {
	f := repoHandler(ctx, repo)

	go tick(ctx, f)
}

func tick(ctx context.Context, handler handler) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			handler(ctx)
			return
		case <-ctx.Done():
			logger.Infof("Cancelled!")
			return
		}
	}
}

func repoHandler(ctx context.Context, repo string) handler {
	return func(c context.Context) {
		now := time.Now()

		timeout, cancel := context.WithTimeout(c, 5*time.Minute)
		defer cancel()

		since, err := fetchFromConsul(ctx, repo)
		if err != nil {
			logger.Error("err", err.Error())
			return
		}

		commits := []*github.RepositoryCommit{}
		opt := &github.CommitsListOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		if since != nil {
			opt.Since = *since
		}

		for {
			r, resp, err := client.Repositories.ListCommits(timeout, *orgName, repo, opt)
			if err != nil {
				if resp.StatusCode == http.StatusConflict {
					err = updateConsul(ctx, repo, now)
					if err != nil {
						logger.Error("err", err.Error())
						return
					}

					return
				}

				logger.Error("err", err.Error())
				return
			}

			commits = append(commits, r...)

			if resp.NextPage == 0 {
				break
			}

			opt.ListOptions.Page = resp.NextPage
		}

		err = updateConsul(ctx, repo, now)
		if err != nil {
			logger.Error("err", err.Error())
			return
		}

		logger.Infof("Repo %s has %d commits!", repo, len(commits))
	}
}
