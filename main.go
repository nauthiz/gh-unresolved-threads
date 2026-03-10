package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type Author struct {
	Login string `json:"login"`
}

type Comment struct {
	URL       string `json:"url"`
	Author    Author `json:"author"`
	Body      string `json:"body"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	CreatedAt string `json:"createdAt"`
}

type Comments struct {
	Nodes []Comment `json:"nodes"`
}

type ReviewThread struct {
	IsResolved bool     `json:"isResolved"`
	Comments   Comments `json:"comments"`
}

type PullRequest struct {
	ReviewThreads struct {
		Nodes []ReviewThread `json:"nodes"`
	} `json:"reviewThreads"`
}

type Repository struct {
	PullRequest PullRequest `json:"pullRequest"`
}

type GraphQLResponse struct {
	Repository Repository `json:"repository"`
}

type PullRequestItem struct {
	Number int `json:"number"`
}

func usage() {
	fmt.Println("Usage: gh-unresolved-comments [flags] <PR_URL | PR_NUMBER | BRANCH>")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -R, --repo [HOST/]OWNER/REPO   Select another repository using the [HOST/]OWNER/REPO format")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gh-unresolved-comments https://github.com/owner/repo/pull/123")
	fmt.Println("  gh-unresolved-comments 123")
	fmt.Println("  gh-unresolved-comments my-feature-branch")
	fmt.Println("  gh-unresolved-comments -R owner/repo 123")
	fmt.Println("  gh-unresolved-comments --repo owner/repo my-feature-branch")
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	// -R / --repo フラグのパース
	var repoFlag string
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-R", "--repo":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires an argument\n", args[i])
				os.Exit(1)
			}
			repoFlag = args[i+1]
			i++
		default:
			remaining = append(remaining, args[i])
		}
	}

	if len(remaining) == 0 {
		usage()
		os.Exit(1)
	}

	arg := remaining[0]

	var owner, repo string
	var prNumber int

	// PR URLかどうか判定
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		var err error
		owner, repo, prNumber, err = parsePRURL(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// リポジトリの特定
		var ghRepo repository.Repository
		if repoFlag != "" {
			var err error
			ghRepo, err = repository.Parse(repoFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing --repo: %v\n", err)
				os.Exit(1)
			}
		} else {
			var err error
			ghRepo, err = repository.Current()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error determining current repository: %v\n", err)
				fmt.Fprintf(os.Stderr, "Use -R / --repo to specify a repository.\n")
				os.Exit(1)
			}
		}
		owner = ghRepo.Owner
		repo = ghRepo.Name

		// PR番号かブランチ名かを判定
		if n, err := strconv.Atoi(arg); err == nil {
			prNumber = n
		} else {
			// ブランチ名からPR番号を取得
			var err error
			prNumber, err = getPRNumberFromBranch(owner, repo, arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	client, err := api.DefaultGraphQLClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating GraphQL client: %v\n", err)
		os.Exit(1)
	}

	query := `query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 100) {
            nodes {
              url
              author { login }
              body
              path
              line
              createdAt
            }
          }
        }
      }
    }
  }
}`

	variables := map[string]any{
		"owner":  owner,
		"repo":   repo,
		"number": prNumber,
	}

	var response GraphQLResponse
	err = client.Do(query, variables, &response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing GraphQL query: %v\n", err)
		os.Exit(1)
	}

	threads := response.Repository.PullRequest.ReviewThreads.Nodes
	var unresolvedThreads []ReviewThread
	for _, thread := range threads {
		if !thread.IsResolved && len(thread.Comments.Nodes) > 0 {
			unresolvedThreads = append(unresolvedThreads, thread)
		}
	}

	if len(unresolvedThreads) == 0 {
		fmt.Println("未解決のコメントは見つかりませんでした。")
		return
	}

	// 概要の出力
	fmt.Printf("## 概要\n\n")
	fmt.Println("| # | URL | 日付 | ファイル | 内容 |")
	fmt.Println("|---|-----|------|----------|------|")

	for i, thread := range unresolvedThreads {
		firstComment := thread.Comments.Nodes[0]
		url := firstComment.URL
		createdAt := ""
		if len(firstComment.CreatedAt) >= 10 {
			createdAt = firstComment.CreatedAt[:10]
		}
		path := firstComment.Path
		body := strings.ReplaceAll(firstComment.Body, "\r\n", " ")
		body = strings.ReplaceAll(body, "\n", " ")
		body = strings.ReplaceAll(body, "\r", "")
		summary := body
		runes := []rune(body)
		if len(runes) > 150 {
			summary = string(runes[:150]) + "..."
		}
		fmt.Printf("| %d | %s | %s | %s | %s |\n", i+1, url, createdAt, path, summary)
	}

	fmt.Printf("\n---\n\n")

	// 各スレッドの詳細
	fmt.Printf("## 未解決コメント\n\n")
	for i, thread := range unresolvedThreads {
		firstComment := thread.Comments.Nodes[0]
		path := firstComment.Path
		url := firstComment.URL

		fmt.Printf("### %d. 未解決のスレッド: %s\n", i+1, path)
		fmt.Printf("URL: %s\n\n", url)

		for _, comment := range thread.Comments.Nodes {
			author := comment.Author.Login
			if author == "" {
				author = "unknown"
			}
			body := comment.Body
			fmt.Printf("**%s**\n", author)
			fmt.Printf("%s\n\n", body)
		}
		fmt.Println("---")
	}
}

func parsePRURL(prURL string) (string, string, int, error) {
	re := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
	matches := re.FindStringSubmatch(prURL)
	if len(matches) != 4 {
		return "", "", 0, fmt.Errorf("invalid PR URL: %s", prURL)
	}

	owner := matches[1]
	repo := matches[2]
	prNumber, err := strconv.Atoi(matches[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number: %v", err)
	}

	return owner, repo, prNumber, nil
}

func getPRNumberFromBranch(owner, repo, branch string) (int, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return 0, fmt.Errorf("creating REST client: %w", err)
	}

	var prs []PullRequestItem
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls?head=%s:%s&state=open", owner, repo, owner, branch), &prs)
	if err != nil {
		return 0, fmt.Errorf("fetching PRs for branch %q: %w", branch, err)
	}

	if len(prs) == 0 {
		return 0, fmt.Errorf("no open pull request found for branch %q in %s/%s", branch, owner, repo)
	}

	return prs[0].Number, nil
}
