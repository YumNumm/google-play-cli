# API 仕様書: `google-play bundles publish`

## コマンド概要

AAB（Android App Bundle）ファイルを Google Play にアップロードし、指定されたトラックにリリースとして公開する。

## API コールフロー

```
┌─────────────────────────────────────────────────────────┐
│ 1. edits.insert      — Edit セッションを作成            │
│ 2. edits.bundles.upload — AAB ファイルをアップロード    │
│ 3. edits.tracks.update  — トラックにリリースを設定      │
│ 4. edits.commit      — Edit をコミット（変更を反映）    │
└─────────────────────────────────────────────────────────┘
```

---

## Step 1: Edit の作成（edits.insert）

### エンドポイント

```
POST https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/edits
```

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `packageName` | string | アプリのパッケージ名（例: `com.example.app`） |

### リクエストボディ

```json
{}
```

空のオブジェクトを送信する。

### レスポンス

```json
{
  "id": "string",
  "expiryTimeSeconds": "string"
}
```

| フィールド | 型 | 説明 |
|-----------|------|------|
| `id` | string | Edit ID（以降の操作で使用） |
| `expiryTimeSeconds` | string | Edit の有効期限（Unix タイムスタンプ） |

### 認証ヘッダー

```
Authorization: Bearer {access_token}
```

---

## Step 2: Bundle のアップロード（edits.bundles.upload）

### エンドポイント

```
POST https://androidpublisher.googleapis.com/upload/androidpublisher/v3/applications/{packageName}/edits/{editId}/bundles
```

> **注意**: ベース URL が `/upload/` プレフィックス付きになる。

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `packageName` | string | アプリのパッケージ名 |
| `editId` | string | Step 1 で取得した Edit ID |

### クエリパラメータ（オプション）

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `deviceTierConfigId` | string | デバイスティア設定 ID |

### リクエスト

- **Content-Type**: `application/octet-stream`
- **Body**: AAB ファイルのバイナリデータ

```
Content-Type: application/octet-stream
Content-Length: {file_size}

<binary AAB data>
```

### レスポンス

```json
{
  "versionCode": 42,
  "sha1": "string",
  "sha256": "string"
}
```

| フィールド | 型 | 説明 |
|-----------|------|------|
| `versionCode` | integer | バンドルのバージョンコード |
| `sha1` | string | SHA-1 ハッシュ |
| `sha256` | string | SHA-256 ハッシュ |

### タイムアウトに関する注意

- Google の推奨: 最低 2 分の HTTP タイムアウト
- codemagic の実装: 10 分（600 秒）のソケットタイムアウトを設定
- AAB ファイルサイズに応じて適切なタイムアウトを設定すること

---

## Step 3: トラックにリリースを設定（edits.tracks.update）

### エンドポイント

```
PUT https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/edits/{editId}/tracks/{track}
```

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `packageName` | string | アプリのパッケージ名 |
| `editId` | string | Edit ID |
| `track` | string | トラック名（例: `production`, `beta`, `alpha`, `internal`） |

### リクエストボディ（Track オブジェクト）

```json
{
  "track": "production",
  "releases": [
    {
      "status": "completed",
      "versionCodes": ["42"],
      "name": "Release v1.0.0",
      "releaseNotes": [
        {
          "language": "en-US",
          "text": "Bug fixes and improvements"
        }
      ],
      "inAppUpdatePriority": 3,
      "userFraction": 0.5
    }
  ]
}
```

### Release オブジェクトのフィールド

| フィールド | 型 | 必須 | 説明 |
|-----------|------|------|------|
| `status` | string | はい | リリースステータス（後述） |
| `versionCodes` | string[] | はい | バージョンコードの配列（文字列） |
| `name` | string | いいえ | リリース名 |
| `releaseNotes` | LocalizedText[] | いいえ | ローカライズされたリリースノート |
| `inAppUpdatePriority` | integer | いいえ | アプリ内更新の優先度（0-5） |
| `userFraction` | double | いいえ | 段階的ロールアウトのユーザー割合（0-1） |

### Release Status の値

| 値 | 説明 | 条件 |
|---|------|------|
| `completed` | 全ユーザーに公開 | デフォルト（draft でも rollout でもない場合） |
| `draft` | ドラフト状態 | `--draft` フラグ指定時 |
| `inProgress` | 段階的ロールアウト中 | `--rollout-fraction` 指定時 |

### LocalizedText オブジェクト

```json
{
  "language": "en-US",
  "text": "Release note content"
}
```

### レスポンス

更新された Track オブジェクト（リクエストと同じ構造）。

---

## Step 4: Edit のコミット（edits.commit）

### エンドポイント

```
POST https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/edits/{editId}:commit
```

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `packageName` | string | アプリのパッケージ名 |
| `editId` | string | Edit ID |

### クエリパラメータ（オプション）

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `changesNotSentForReview` | boolean | `true` にすると、変更はレビューに送信されない |

### リクエストボディ

なし（空）。

### レスポンス

```json
{
  "id": "string",
  "expiryTimeSeconds": "string"
}
```

---

## CLI パラメータと API パラメータの対応表

| CLI パラメータ | API パラメータ | 使用箇所 |
|---------------|---------------|---------|
| `--bundle, -b` | `media_body` (multipart upload) | edits.bundles.upload |
| `--track, -t` | `track` (path param) | edits.tracks.update |
| `--release-name, -r` | `releases[].name` | edits.tracks.update body |
| `--in-app-update-priority, -i` | `releases[].inAppUpdatePriority` | edits.tracks.update body |
| `--rollout-fraction, -f` | `releases[].userFraction` + `status: "inProgress"` | edits.tracks.update body |
| `--draft, -d` | `releases[].status: "draft"` | edits.tracks.update body |
| `--release-notes, -n` | `releases[].releaseNotes` | edits.tracks.update body |
| `--changes-not-sent-for-review` | `changesNotSentForReview` (query param) | edits.commit |
| `--credentials` | — | 認証時に使用（API 呼び出し時は Bearer トークンに変換） |

## パッケージ名の取得

codemagic の実装では、AAB ファイルからパッケージ名を自動抽出している（`bundletool` の `dump resources` を使用）。
自作 CLI で実装する場合は、以下のいずれかの方法を取る:

1. `bundletool` を利用して AAB からパッケージ名を抽出
2. CLI パラメータとして明示的に指定させる（よりシンプル）
3. AAB の ZIP 構造を解析して `AndroidManifest.xml` からパッケージ名を読み取る

## エラーハンドリング

- codemagic の実装ではリトライ回数 3 回（`num_retries=3`）
- OAuth2 エラー、HTTP エラー、その他の Google API エラーを種別ごとにハンドリング
- Edit は明示的にコミットしないと変更が反映されない（安全性の担保）
