# Git Command Introduction
Preparation: If you don't have a GitHub account, you need to create one before proceeding to the next step.

## 1 Fork the Code
1. Visit https://github.com/thinkinaixyz/go-mcp
2. Click the "Fork" button (located at the top right of the page)

## 2 Clone the Code
We generally recommend setting the origin as the official repository and setting up your own upstream.

If you have enabled SSH on GitHub, we recommend using SSH; otherwise, use HTTPS. The difference between the two is that when using HTTPS, you need to enter authentication information every time you push code to the remote repository.
We strongly recommend always using HTTPS for the official repository to avoid accidental operations.

```bash
git clone https://github.com/thinkinaixyz/go-mcp.git
cd go-mcp
git remote add upstream 'git@github.com:<your github username>/go-mcp.git'
```
You can replace "upstream" with any name you like, such as your username, nickname, or simply "me". Remember to make corresponding replacements in subsequent commands.

## 3 Sync the Code
Unless you've just cloned the code locally, we need to sync the remote repository's code first.
git fetch

When not specifying a remote repository, this command will only sync the origin's code. If we need to sync our forked repository, we can add the remote repository name:
git fetch upstream

## 4 Create a Feature Branch
When creating a new feature branch, we need to first consider which branch to branch from.
Let's assume we want our new feature to be merged into the `main` branch, or that our new feature should be based on `main`, execute:
```bash
git checkout -b feature/my-feature origin/main
```
This creates a branch that is identical to the code on `origin/main`.

## 5 Golint
```bash
golint $(go list ./... | grep -v /examples/)
golangci-lint run $(go list ./... | grep -v /examples/)
```

## 6 Go Test
```bash
go test -v -race $(go list ./... | grep -v /examples/) -coverprofile=coverage.txt -covermode=atomic
```

## 7 Submit Commit
```bash
git add .
git commit
git push upstream my-feature
```

## 8 Submit PR
Visit https://github.com/thinkinaixyz/go-mcp,
Click "Compare" to compare changes and click "Pull request" to submit the PR
