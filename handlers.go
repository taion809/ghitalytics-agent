package main

import (
	"context"
	"io"
	"time"

	"bytes"

	"encoding/json"

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

		commits, err := client.Commits(timeout, repo, since)
		if err != nil {
			logger.Error("err", err.Error())
			return
		}

		lines, err := commitsToLines(commits)
		if err != nil {
			logger.Error("err", err.Error())
			return
		}

		err = storage.Append("/commits/"+repo+".log", lines)
		if err != nil {
			logger.Error("err", err.Error())
			return
		}

		err = updateConsul(ctx, repo, now)
		if err != nil {
			logger.Error("err", err.Error())
			return
		}

		logger.Infof("Repo %s has %d commits!", repo, len(commits))
	}
}

func commitsToLines(commits []*github.RepositoryCommit) (io.Reader, error) {
	buf := &bytes.Buffer{}
	for _, v := range commits {
		commit := v
		j, err := json.Marshal(commit)
		if err != nil {
			return nil, err
		}

		buf.Write(j)
		buf.WriteByte('\n')
	}

	return buf, nil
}
