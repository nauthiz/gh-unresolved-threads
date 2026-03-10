# gh-unresolved-threads

GitHub Pull Request の未解決レビュースレッドを一覧表示するツールです。

## 概要

指定した PR の未解決（Unresolved）レビュースレッドを取得し、テーブル形式（デフォルト）または Markdown 形式で標準出力に表示します。

## インストール

### 前提条件

- [Go](https://golang.org/) 1.25 以上
- [GitHub CLI (`gh`)](https://cli.github.com/) がインストール済みで、`gh auth login` で認証済みであること

### ビルド

```bash
$ git clone https://github.com/nauthiz/gh-unresolved-threads.git
$ cd gh-unresolved-threads
$ go build -o gh-unresolved-threads .
```

ビルドされた `gh-unresolved-threads` をインストール。

```bash
$ gh extension install .
```

## 使い方

```
gh-unresolved-threads [flags] <PR_URL | PR_NUMBER | BRANCH>
```

### フラグ

| フラグ | 説明 |
|--------|------|
| `-R`, `--repo [HOST/]OWNER/REPO` | リポジトリを明示的に指定します |
| `--markdown` | Markdown 形式で出力します（デフォルト: テーブル形式） |

### 引数

| 形式 | 説明 |
|------|------|
| `PR_URL` | PR の URL（例: `https://github.com/owner/repo/pull/123`） |
| `PR_NUMBER` | PR 番号（例: `123`）。カレントディレクトリまたは `--repo` で指定したリポジトリを使用 |
| `BRANCH` | ブランチ名（例: `my-feature-branch`）。そのブランチに紐づくオープンな PR を検索 |

### 実行例

```bash
# PR URL で指定
$ gh-unresolved-threads https://github.com/owner/repo/pull/123

# PR 番号で指定（カレントディレクトリのリポジトリを自動検出）
$ gh-unresolved-threads 123

# ブランチ名で指定（カレントディレクトリのリポジトリを自動検出）
$ gh-unresolved-threads my-feature-branch

# --repo でリポジトリを明示して PR 番号を指定
$ gh-unresolved-threads -R owner/repo 123

# --repo でリポジトリを明示してブランチ名を指定
$ gh-unresolved-threads --repo owner/repo my-feature-branch
```

### 出力例（テーブル形式・デフォルト）

```
1  https://github.com/.../pull/...#discussion_r...
   コメントの内容...

2  https://github.com/.../pull/...#discussion_r...
   別のコメントの内容...
```

### 出力例（Markdown 形式・`--markdown` 指定時）

```
## 概要

| # | 投稿者 | URL | 内容 |
|---|--------|-----|------|
| 1 | reviewer | https://github.com/... | コメントの内容... |

---

## 未解決コメント

### 1. 未解決のスレッド: main.go
URL: https://github.com/...

**reviewer**
コメントの内容

---
```

## ライセンス

MIT
