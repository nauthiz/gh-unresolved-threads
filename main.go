package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gh-unresolved-comments <PR_URL>")
		os.Exit(1)
	}

	prURL := os.Args[1]
	owner, repo, prNumber, err := parsePRURL(prURL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	client, err := api.DefaultGraphQLClient()
	if err != nil {
		fmt.Printf("Error creating GraphQL client: %v\n", err)
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
		fmt.Printf("Error executing GraphQL query: %v\n", err)
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
