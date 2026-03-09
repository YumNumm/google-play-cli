# 調査サマリー: Google Play CLI コマンドの内部実装と Workload Identity Federation 対応

## 調査対象

codemagic-ci-cd/cli-tools の以下2つのコマンドが内部でどのような Google Play API を叩いているかを調査し、
Workload Identity Federation (WIF) 対応の別 CLI ツールを作成する実現可能性を検討する。

1. `google-play bundles publish` — AAB を Google Play にアップロードし、指定トラックにリリースとして公開
2. `google-play get-latest-build-number` — Google Play から最新のビルド番号（versionCode）を取得

## 調査結果の概要

### 使用されている Google API

- **API 名**: Google Play Android Publisher API
- **バージョン**: v3
- **ベース URL**: `https://androidpublisher.googleapis.com`
- **必要な OAuth スコープ**: `https://www.googleapis.com/auth/androidpublisher`

### 現在の認証方式（codemagic cli-tools）

- **ライブラリ**: `oauth2client.service_account.ServiceAccountCredentials`
- **方式**: サービスアカウント JSON キーファイルを直接渡す（`from_json_keyfile_dict`）
- **問題点**: 長寿命のサービスアカウントキーをシークレットとして管理する必要がある

### Workload Identity Federation 対応の実現可能性

**結論: 実現可能**

理由:

1. Google Play Android Publisher API は標準的な Google Cloud OAuth 2.0 認証を使用している
2. WIF は OAuth 2.0 トークン交換（RFC 8693）に基づいており、最終的に標準の Bearer アクセストークンを取得する
3. アクセストークンさえ取得できれば、REST API は同じように呼び出せる
4. Google の公式 Auth ライブラリ（各言語向け）は WIF をサポートしている

## ファイル構成

| ファイル | 内容 |
|---------|------|
| `02-api-spec-bundles-publish.md` | `bundles publish` コマンドの API 仕様 |
| `03-api-spec-get-latest-build-number.md` | `get-latest-build-number` コマンドの API 仕様 |
| `04-authentication-and-wif.md` | 認証方式と WIF 対応の詳細設計 |
