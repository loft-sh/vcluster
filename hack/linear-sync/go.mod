module github.com/loft-sh/linear-sync

go 1.23.0

require (
	github.com/loft-sh/changelog v0.0.0-00010101000000-000000000000
	github.com/shurcooL/githubv4 v0.0.0-20240120211514-18a1ae0e79dc
	github.com/shurcooL/graphql v0.0.0-20230722043721-ed46e5a46466
	golang.org/x/oauth2 v0.27.0
)

require (
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/google/go-github/v59 v59.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
)

replace github.com/loft-sh/changelog => ../changelog
