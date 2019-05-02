# hookeye

## Usage

Running `make` will build the binary to `BUILD/hookeye`.

To start HTTP server as following:

```
$ env GITHUB_TOKEN=<oauth_token> GITHUB_SECRET=<secret> ./BUILD/hookeye
```

See `hookeye -help` for command line flags.

## Hooks

### Attach project card to new issues

Send [Github's "issues" webhook][1] to `/github` to attach created issue to project.

TODO: "repository to project" mapping is hardcoded in `hooks/issues_processor.go`.

[1]: https://developer.github.com/webhooks/
