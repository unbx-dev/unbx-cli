# Unbx CLI — GitHub Action

A GitHub Action that scans PR diffs for architecture policy violations and posts inline fix suggestions as review comments.

## How it works

1. Fetches changed files from the PR diff
2. Parses each file with tree-sitter and generates an anonymized AST (no raw source code is sent)
3. Sends the anonymized AST to the unbx backend API for scanning
4. If violations are found, posts inline review comments with suggested fixes to the PR
5. Fails the CI pipeline when violations are detected (exit code 1)

## Setup

### 1. Add secrets

Go to **Settings > Secrets and variables > Actions** in your GitHub repository and add the following secrets.

| Secret | Description |
|---|---|
| `UNBX_REPOSITORY_ID` | Your repository ID from the unbx dashboard |
| `UNBX_ACCESS_TOKEN` | Your unbx API key |
| `UNBX_SECRET_TOKEN` | Your unbx secret key |
| `UNBX_API_URL` | unbx backend API URL (e.g. `https://api.unbx.dev`) |

### 2. Add a workflow

Create `.github/workflows/unbx-scan.yml`:

```yaml
name: unbx Code Scan

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  pull-requests: write  # required to post review comments

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: unbx Code Scan
        uses: your-org/unbx-cli@v1
        with:
          repository_id: ${{ secrets.UNBX_REPOSITORY_ID }}
          access_token:  ${{ secrets.UNBX_ACCESS_TOKEN }}
          secret_token:  ${{ secrets.UNBX_SECRET_TOKEN }}
          synk_api_url:  ${{ secrets.UNBX_API_URL }}
```

## Action inputs

| Input | Required | Description |
|---|---|---|
| `repository_id` | ✅ | Repository ID from the unbx dashboard |
| `access_token` | ✅ | unbx API key |
| `secret_token` | ✅ | unbx secret key |
| `synk_api_url` | ✅ | unbx backend API URL |
| `github_token` | | GitHub token for posting review comments (defaults to `github.token`) |

## Supported languages

| Language | Extensions |
|---|---|
| Go | `.go` |
| JavaScript | `.js` `.jsx` `.mjs` `.cjs` |
| TypeScript | `.ts` |
| TSX | `.tsx` |
| Python | `.py` |
| Ruby | `.rb` |
| Rust | `.rs` |
| Java | `.java` |
| PHP | `.php` |
| C | `.c` `.h` |
| C++ | `.cpp` `.cc` `.cxx` `.hpp` |
| Bash | `.sh` `.bash` |

Files with unsupported extensions are skipped.

## Running locally

Set the required environment variables and run directly:

```bash
export REPOSITORY_ID=your-repository-id
export ACCESS_TOKEN=your-access-token
export SECRET_TOKEN=your-secret-token
export SYNK_API_URL=https://api.unbx.dev
export GITHUB_TOKEN=your-github-token
export REPO_SLUG=owner/repo
export PR_NUMBER=123
export SYNK_COMMIT_SHA=abc1234

go run main.go
```

## Privacy

No source code is transmitted. Only the **anonymized AST** — a structural representation with all identifiers hashed — is sent to the API.
