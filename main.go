package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adjust/hookeye/github"
	"github.com/adjust/hookeye/hooks"
	"github.com/adjust/hookeye/hooks/githubsvc"
	"github.com/adjust/hookeye/stream"
	"github.com/machinebox/graphql"
	"github.com/peterbourgon/ff"
)

const defaultGitHubAPIEndpoint = "https://api.github.com/graphql"

const githubIssuesTopic = "github/issues"

type Config struct {
	Addr        string
	ExitTimeout time.Duration

	StreamCompactInterval time.Duration

	GithubAPIEndpoint   string
	GithubClientTimeout time.Duration
	GithubToken         string
	GithubSecret        string
}

func main() {
	var conf Config

	flag.StringVar(&conf.Addr, "addr", ":10080", "address to listen")
	flag.DurationVar(&conf.ExitTimeout, "exit-timeout", 5*time.Second, "exit timeout")

	flag.DurationVar(&conf.StreamCompactInterval, "stream.compact-interval", time.Minute, "stream compaction interval")

	flag.StringVar(&conf.GithubAPIEndpoint, "github.api-endpoint", defaultGitHubAPIEndpoint, "github api graphql endpoint")
	flag.DurationVar(&conf.GithubClientTimeout, "github.client.timeout", 0, "github api client request timeout")

	// TODO(narqo): parse config from file
	ff.Parse(flag.CommandLine, os.Args[1:])

	// read github token from env
	conf.GithubToken = os.Getenv("GITHUB_TOKEN")
	if conf.GithubToken == "" {
		log.Fatal("env: no GITHUB_TOKEN")
	}

	// read github secret from env (see https://developer.github.com/webhooks/securing/)
	conf.GithubSecret = os.Getenv("GITHUB_SECRET")

	if err := run(context.Background(), conf); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, conf Config) error {
	stream := stream.New()

	if conf.StreamCompactInterval > 0 {
		go func() {
			tick := time.Tick(conf.StreamCompactInterval)
			for range tick {
				stream.Compact()
			}
		}()
	}

	httpClient := &http.Client{
		Timeout: conf.GithubClientTimeout,
	}
	githubSvc := &githubsvc.Service{
		Client: github.NewClient(conf.GithubAPIEndpoint, conf.GithubToken, graphql.WithHTTPClient(httpClient)),
	}
	issuesProcessor := &hooks.IssuesProcessor{
		GithubService: githubSvc,
	}
	stream.SubscribeN(githubIssuesTopic, issuesProcessor, 2)

	mux := http.NewServeMux()

	githubHandler := NewGithubHandler(stream, conf.GithubSecret)
	githubHandler.RegisterRoutes(mux)

	server := http.Server{
		Addr:    conf.Addr,
		Handler: mux,
	}

	errc := make(chan error, 1)
	go func() {
		log.Printf("server is listening on %s\n", server.Addr)
		errc <- server.ListenAndServe()
	}()

	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		log.Printf("exiting %v\n", sig)
	case err := <-errc:
		if err != http.ErrServerClosed {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, conf.ExitTimeout)
	defer cancel()

	return server.Shutdown(ctx)
}
