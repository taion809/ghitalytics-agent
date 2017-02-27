package main

import (
	"context"
	"log"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"os/signal"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

var (
	ghAccessKey = kingpin.Flag("github.access-key", "Github oauth2 access key").OverrideDefaultFromEnvar("ENV_GH_ACCESS_TOKEN").String()
	consulAddr  = kingpin.Flag("consul.addr", "Consul KV api address (ex: localhost:8500)").OverrideDefaultFromEnvar("ENV_CONSUL_ADDR").String()
	orgName     = kingpin.Arg("github.organization-name", "Github organization name").Required().String()
	logger      *zap.SugaredLogger
	bgctx       context.Context
	client      *GithubApi
	consul      *api.Client
)

func init() {
	bgctx = context.Background()

	l, err := zap.NewProduction()
	if err != nil {
		log.Fatalln(err)
	}

	logger = l.Sugar()
}

func main() {
	kingpin.Parse()

	var err error
	ctx, cancel := context.WithCancel(bgctx)
	defer cancel()

	client = NewDefaultGithubApi(ctx, *orgName, *ghAccessKey)

	consul, err = api.NewClient(&api.Config{Address: "localhost:8500"})
	if err != nil {
		logger.Fatalf("err: %q", err.Error())
	}

	repos, err := client.ListRepositories(ctx)
	if err != nil {
		logger.Fatalf("err: %q", err.Error())
	}

	logger.Infof("Len: %d", len(repos))
	for _, v := range repos {
		register(ctx, v)
	}

	go gracefulShutdown(cancel)
}

func gracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)
	<-c

	cancel()
	os.Exit(0)
}

func fetchAndSave(key string) {
	// fech object from s3
	// append
	// write object to s3
}
