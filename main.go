package main

import (
	"context"
	"log"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"os/signal"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

var (
	ghAccessKey    = kingpin.Flag("github.access-key", "Github oauth2 access key").OverrideDefaultFromEnvar("ENV_GH_ACCESS_TOKEN").String()
	consulAddr     = kingpin.Flag("consul.addr", "Consul KV api address (ex: localhost:8500)").OverrideDefaultFromEnvar("ENV_CONSUL_ADDR").String()
	storageType    = kingpin.Flag("storage.type", "Storage type (ex: os, s3)").OverrideDefaultFromEnvar("ENV_STORAGE_TYPE").Default("os").String()
	storageBase    = kingpin.Flag("storage.root", "Root directory for storage").OverrideDefaultFromEnvar("ENV_STORAGE_ROOT").Required().String()
	storageProfile = kingpin.Flag("aws.profile", "Profile to use for S3").OverrideDefaultFromEnvar("AWS_PROFILE").Default("default").String()
	orgName        = kingpin.Arg("github.organization-name", "Github organization name").Required().String()
	logger         *zap.SugaredLogger
	bgctx          context.Context
	client         *GithubApi
	consul         *api.Client
	storage        Storage
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
	storage = makeStorage()

	consul, err = api.NewClient(&api.Config{Address: *consulAddr})
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

	gracefulShutdown(cancel)
}

func gracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)
	<-c

	cancel()
	os.Exit(0)
}

func makeStorage() Storage {
	if *storageType == "s3" {
		awsSession := session.Must(session.NewSessionWithOptions(session.Options{
			Profile: *storageProfile,
		}))

		return NewS3Storage(*storageBase, awsSession)
	}

	return &FsStorage{Base: *storageBase}
}
