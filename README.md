# [UNDER DEVELOPMENT] google-play-cli

Google Play Android Publisher API v3 を操作する CLI ツール。
[codemagic cli-tools](https://github.com/codemagic-ci-cd/cli-tools) の `google-play` コマンドの代替として、Workload Identity Federation (WIF) をネイティブサポートする。

## インストール

```bash
go install github.com/YumNumm/google-play-cli@latest
```

または、ソースからビルド:

```bash
go build -o google-play-cli .
```

## 認証

2 つの認証方式をサポートする。

### 1. サービスアカウント JSON 文字列 (`--credentials`)

`--credentials` フラグにサービスアカウント JSON を文字列として渡す:

```bash
google-play-cli bundles publish \
  --credentials '{"type":"service_account","project_id":"...","private_key":"...","client_email":"..."}' \
  --package-name com.example.app \
  --bundle app.aab \
  --track internal
```

### 2. Application Default Credentials (ADC)

`--credentials` を省略すると、[Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) を使用する。
以下の順序で認証情報を検索する:

1. `GOOGLE_APPLICATION_CREDENTIALS` 環境変数が指すファイル
   - `type: "service_account"` → サービスアカウントキー
   - `type: "external_account"` → Workload Identity Federation
2. GCE/Cloud Run 等のメタデータサーバ
3. `gcloud auth application-default login` の認証情報

### GitHub Actions での WIF 利用

[google-github-actions/auth](https://github.com/google-github-actions/auth) と組み合わせると、
サービスアカウントキーなしで Google Play にアクセスできる:

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write

    steps:
      - uses: actions/checkout@v4

      - uses: google-github-actions/auth@v3
        with:
          workload_identity_provider: 'projects/123456789/locations/global/workloadIdentityPools/pool/providers/provider'
          service_account: 'sa@project.iam.gserviceaccount.com'

      - run: |
          google-play-cli bundles publish \
            --package-name com.example.app \
            --bundle app/build/outputs/bundle/release/app-release.aab \
            --track internal
```

## コマンド

### `bundles publish`

AAB ファイルを Google Play にアップロードし、指定トラックにリリースする。

```bash
google-play-cli bundles publish [flags]
```

**必須フラグ:**

| フラグ | 短縮 | 説明 |
|--------|------|------|
| `--package-name` | `-p` | アプリのパッケージ名 |
| `--bundle` | `-b` | AAB ファイルのパス |
| `--track` | `-t` | トラック名 (`production`, `beta`, `alpha`, `internal`) |

**オプションフラグ:**

| フラグ | 短縮 | 説明 |
|--------|------|------|
| `--credentials` | | サービスアカウント JSON 文字列 (省略時は ADC) |
| `--release-name` | `-r` | リリース名 |
| `--release-notes` | `-n` | リリースノート (JSON 文字列) |
| `--in-app-update-priority` | `-i` | アプリ内更新の優先度 (0-5) |
| `--rollout-fraction` | `-f` | 段階的ロールアウトの割合 (0.0-1.0) |
| `--draft` | `-d` | ドラフトとしてリリース |
| `--changes-not-sent-for-review` | | レビューに送信しない |

**`--release-notes` の形式:**

```bash
--release-notes '[{"language":"en-US","text":"Bug fixes"},{"language":"ja-JP","text":"バグ修正"}]'
```

**使用例:**

```bash
# 内部テストトラックにアップロード
google-play-cli bundles publish \
  -p com.example.app \
  -b app.aab \
  -t internal

# ドラフトリリース
google-play-cli bundles publish \
  -p com.example.app \
  -b app.aab \
  -t production \
  -d \
  -r "v1.0.0" \
  -n '[{"language":"en-US","text":"Initial release"}]'

# 段階的ロールアウト (50%)
google-play-cli bundles publish \
  -p com.example.app \
  -b app.aab \
  -t production \
  -f 0.5
```

### `get-latest-build-number`

Google Play から最新のビルド番号 (versionCode) を取得する。

```bash
google-play-cli get-latest-build-number [flags]
```

**必須フラグ:**

| フラグ | 短縮 | 説明 |
|--------|------|------|
| `--package-name` | `-p` | アプリのパッケージ名 |

**オプションフラグ:**

| フラグ | 短縮 | 説明 |
|--------|------|------|
| `--credentials` | | サービスアカウント JSON 文字列 (省略時は ADC) |
| `--tracks` | `-t` | フィルタ対象トラック (カンマ区切りまたは複数指定) |

**使用例:**

```bash
# 全トラックから最新のビルド番号を取得
google-play-cli get-latest-build-number -p com.example.app

# 特定のトラックのみ
google-play-cli get-latest-build-number -p com.example.app -t production,beta

# CI/CD でのビルド番号インクリメント
CURRENT=$(google-play-cli get-latest-build-number -p com.example.app)
NEXT=$((CURRENT + 1))
echo "Next build number: $NEXT"
```

## 開発

```bash
# テスト実行
go test ./...

# カバレッジ確認
go test ./internal/... ./cmd/ -coverprofile=coverage.out
go tool cover -func=coverage.out

# ビルド
go build -o google-play-cli .
```

## ライセンス

MIT
