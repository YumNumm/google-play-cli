# 認証方式と Workload Identity Federation 対応の詳細

## 1. 現在の認証方式（codemagic cli-tools）

### 概要

codemagic cli-tools は `oauth2client` ライブラリの `ServiceAccountCredentials.from_json_keyfile_dict()` を使用して、
サービスアカウントの JSON キーファイルから直接認証を行っている。

### フロー

```
サービスアカウント JSON キー
    ↓
ServiceAccountCredentials.from_json_keyfile_dict(json_dict)
    ↓
discovery.build("androidpublisher", "v3", credentials=credentials)
    ↓
Google Play Android Publisher API 呼び出し
```

### サービスアカウント JSON キーの構造

```json
{
  "type": "service_account",
  "project_id": "your-project-id",
  "private_key_id": "key-id",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----\n",
  "client_email": "sa@your-project.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/..."
}
```

### 問題点

1. **セキュリティリスク**: 長寿命のプライベートキーをシークレットとして管理する必要がある
2. **キーローテーション**: 手動でのキーローテーションが必要
3. **漏洩リスク**: キーが漏洩した場合、失効させるまでアクセスされ続ける

---

## 2. Workload Identity Federation（WIF）

### 概要

Workload Identity Federation は、外部のワークロードがサービスアカウントキーなしで
Google Cloud リソースにアクセスするための仕組み。OAuth 2.0 トークン交換（RFC 8693）に基づく。

### 対応する外部 ID プロバイダ

| プロバイダ | トークン種別 |
|-----------|-------------|
| GitHub Actions | OIDC JWT |
| GitLab CI/CD | OIDC JWT |
| AWS | STS トークン |
| Azure | マネージド ID トークン |
| Kubernetes (GKE 以外) | サービスアカウントトークン |
| 任意の OIDC プロバイダ | OIDC JWT |
| 任意の SAML プロバイダ | SAML アサーション |

### トークン交換フロー

```
┌──────────────────┐    ①外部トークン取得     ┌──────────────────┐
│  外部ワークロード │ ◄───────────────────── │  ID プロバイダ    │
│  (CI/CD 等)      │                         │  (GitHub, etc.)  │
└──────┬───────────┘                         └──────────────────┘
       │
       │ ②外部トークンを送信
       ▼
┌──────────────────┐
│  Google STS      │  POST https://sts.googleapis.com/v1/token
│  エンドポイント  │
└──────┬───────────┘
       │
       │ ③フェデレーションアクセストークンを返却
       ▼
┌──────────────────┐
│  (オプション)    │  POST https://iamcredentials.googleapis.com/v1/
│  SA インパーソ   │    projects/-/serviceAccounts/{SA_EMAIL}:generateAccessToken
│  ネーション      │
└──────┬───────────┘
       │
       │ ④SA アクセストークンを返却
       ▼
┌──────────────────┐
│  Google Play API │  Authorization: Bearer {access_token}
└──────────────────┘
```

### Step ①: 外部トークンの取得

各 CI/CD 環境の仕組みに依存する。例:

**GitHub Actions の場合:**

```yaml
- uses: google-github-actions/auth@v2
  with:
    workload_identity_provider: 'projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID'
    service_account: 'SA_NAME@PROJECT_ID.iam.gserviceaccount.com'
```

GitHub Actions は自動で OIDC JWT を取得してくれる。CLI ツールとして使う場合は、
環境変数や設定ファイルから外部トークンを受け取る形にする。

### Step ②: STS トークン交換

#### エンドポイント

```
POST https://sts.googleapis.com/v1/token
```

#### リクエスト（application/x-www-form-urlencoded）

| パラメータ | 値 | 説明 |
|-----------|------|------|
| `grant_type` | `urn:ietf:params:oauth:grant-type:token-exchange` | 固定値 |
| `audience` | `//iam.googleapis.com/projects/{PROJECT_NUMBER}/locations/global/workloadIdentityPools/{POOL_ID}/providers/{PROVIDER_ID}` | WIF プロバイダのリソース名 |
| `scope` | `https://www.googleapis.com/auth/cloud-platform` | 要求するスコープ |
| `requested_token_type` | `urn:ietf:params:oauth:token-type:access_token` | 要求するトークン種別 |
| `subject_token_type` | `urn:ietf:params:oauth:token-type:jwt` | 外部トークンの種別（OIDC の場合） |
| `subject_token` | `{外部 OIDC トークン}` | 外部 ID プロバイダから取得したトークン |

#### レスポンス

```json
{
  "access_token": "ya29.c...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

> **重要**: この `access_token` はフェデレーションアクセストークンであり、
> Google Play API が直接受け付けない場合がある。その場合は Step ③ が必要。

### Step ③: サービスアカウント インパーソネーション（推奨）

Google Play Android Publisher API はフェデレーションアクセストークンを直接サポートしない場合があるため、
サービスアカウントのインパーソネーションを通じて通常のアクセストークンを取得する。

#### エンドポイント

```
POST https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/{SERVICE_ACCOUNT_EMAIL}:generateAccessToken
```

#### リクエスト

```json
{
  "scope": [
    "https://www.googleapis.com/auth/androidpublisher"
  ],
  "lifetime": "3600s"
}
```

#### 認証ヘッダー

```
Authorization: Bearer {federated_access_token}
```

Step ② で取得したフェデレーションアクセストークンを使用。

#### レスポンス

```json
{
  "accessToken": "ya29.c...",
  "expireTime": "2025-01-01T00:00:00Z"
}
```

この `accessToken` を使って Google Play API を呼び出す。

### Step ④: Google Play API 呼び出し

取得したアクセストークンを Bearer トークンとして使用:

```
Authorization: Bearer {access_token}
```

---

## 3. WIF の前提条件（Google Cloud 側の設定）

### 必要なリソース

1. **Workload Identity Pool** の作成
2. **Workload Identity Pool Provider** の作成（OIDC/SAML/AWS 等）
3. **サービスアカウント** の作成（Google Play Console 権限付き）
4. **IAM バインディング**: 外部 ID → サービスアカウントのインパーソネーション権限

### IAM 設定例（GitHub Actions の場合）

```bash
# Workload Identity Pool の作成
gcloud iam workload-identity-pools create "github-pool" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --display-name="GitHub Actions Pool"

# Provider の作成
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --workload-identity-pool="github-pool" \
  --display-name="GitHub Provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository" \
  --issuer-uri="https://token.actions.githubusercontent.com"

# サービスアカウントへのインパーソネーション権限付与
gcloud iam service-accounts add-iam-policy-binding "${SA_EMAIL}" \
  --project="${PROJECT_ID}" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github-pool/attribute.repository/${GITHUB_REPO}"
```

---

## 4. 各言語での WIF サポート状況

### 公式 Auth ライブラリの WIF 対応

| 言語 | ライブラリ | WIF サポート | 備考 |
|------|-----------|-------------|------|
| Rust | `google-cloud-auth` / `gcloud-sdk` | あり | `ExternalAccountCredentials` |
| Go | `golang.org/x/oauth2/google` | あり | `google.CredentialsFromJSON` で WIF 設定ファイルを読み込み可能 |
| Dart | `googleapis_auth` | 限定的 | サービスアカウントキーのみ直接サポート。WIF は手動実装が必要 |
| TypeScript | `google-auth-library` | あり | `ExternalAccountClient` クラス |

### WIF 設定ファイル（Credential Configuration File）

各言語の公式ライブラリは以下の形式の JSON 設定ファイルを読み込める:

```json
{
  "type": "external_account",
  "audience": "//iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID",
  "subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
  "token_url": "https://sts.googleapis.com/v1/token",
  "credential_source": {
    "file": "/path/to/oidc/token",
    "format": {
      "type": "text"
    }
  },
  "service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/SA@PROJECT.iam.gserviceaccount.com:generateAccessToken",
  "service_account_impersonation": {
    "token_lifetime_seconds": 3600
  }
}
```

この設定ファイルは `gcloud iam workload-identity-pools create-cred-config` コマンドで生成できる。

---

## 5. CLI ツールでの認証設計（推奨）

### 認証モード

CLI ツールでは以下の 2 つの認証モードをサポートすることを推奨:

#### モード 1: サービスアカウントキー（従来方式）

```bash
my-cli --credentials @file:service-account.json bundles publish ...
```

#### モード 2: WIF（Credential Configuration File）

```bash
my-cli --credentials @file:wif-config.json bundles publish ...
```

#### モード 3: アクセストークン直接指定

```bash
my-cli --access-token "${ACCESS_TOKEN}" bundles publish ...
```

### 認証の判別ロジック

```
credentials JSON を読み込み
    ↓
"type" フィールドを確認
    ↓
├── "service_account"     → サービスアカウントキー認証
├── "external_account"    → Workload Identity Federation
└── それ以外              → エラー
```

### REST API を直接叩く場合の認証実装

Google のクライアントライブラリ（discovery ベース等）を使わずに REST API を直接叩く場合、
認証は単にアクセストークンを取得して `Authorization: Bearer {token}` ヘッダーに設定するだけ。

WIF の場合のトークン取得手順:

1. 外部トークンを取得（環境依存）
2. STS エンドポイントでトークン交換
3. （必要に応じて）サービスアカウントインパーソネーション
4. 取得したアクセストークンを API リクエストに付与

---

## 6. 実現可能性のまとめ

| 項目 | 評価 | 理由 |
|------|------|------|
| 技術的実現可能性 | ✅ 可能 | REST API は Bearer トークンのみ必要。認証方式は独立 |
| Go での実装 | ✅ 容易 | `golang.org/x/oauth2/google` が WIF をフルサポート |
| Rust での実装 | ✅ 可能 | `google-cloud-auth` クレートで WIF 対応可能 |
| TypeScript での実装 | ✅ 容易 | `google-auth-library` が `ExternalAccountClient` を提供 |
| Dart での実装 | ⚠️ やや困難 | 公式ライブラリの WIF サポートが限定的。STS トークン交換を手動実装する必要がある可能性 |

### 推奨言語

- **Go**: Google Cloud エコシステムとの親和性が高く、シングルバイナリで配布しやすい
- **TypeScript**: npm パッケージとして配布しやすく、`google-auth-library` の WIF サポートが充実
- **Rust**: パフォーマンスが必要な場合に適切。ただし Google Cloud 関連のエコシステムは Go/TS ほど成熟していない
