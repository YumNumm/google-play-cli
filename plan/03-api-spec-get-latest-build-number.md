# API 仕様書: `google-play get-latest-build-number`

## コマンド概要

Google Play の指定パッケージから、最新のビルド番号（versionCode）を取得する。
トラックを指定しない場合は全トラックから最大値を返す。

## API コールフロー

```
┌─────────────────────────────────────────────────────────┐
│ 1. edits.insert       — Edit セッションを作成           │
│ 2. edits.tracks.list  — 全トラック情報を取得            │
│ 3. edits.delete       — Edit セッションを削除（読取専用）│
└─────────────────────────────────────────────────────────┘
```

> **ポイント**: このコマンドは読み取り専用のため、`commit` ではなく `delete` で Edit を破棄する。

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

### レスポンス

```json
{
  "id": "string",
  "expiryTimeSeconds": "string"
}
```

---

## Step 2: トラック一覧の取得（edits.tracks.list）

### エンドポイント

```
GET https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/edits/{editId}/tracks
```

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `packageName` | string | アプリのパッケージ名 |
| `editId` | string | Step 1 で取得した Edit ID |

### リクエストボディ

なし。

### レスポンス

```json
{
  "kind": "androidpublisher#tracksListResponse",
  "tracks": [
    {
      "track": "production",
      "releases": [
        {
          "name": "Release 1.0",
          "versionCodes": ["100", "101"],
          "status": "completed",
          "releaseNotes": [
            {
              "language": "en-US",
              "text": "Initial release"
            }
          ]
        },
        {
          "name": "Release 1.1",
          "versionCodes": ["110"],
          "status": "draft"
        }
      ]
    },
    {
      "track": "beta",
      "releases": [
        {
          "versionCodes": ["120"],
          "status": "completed"
        }
      ]
    },
    {
      "track": "alpha",
      "releases": []
    }
  ]
}
```

### Track オブジェクト

| フィールド | 型 | 説明 |
|-----------|------|------|
| `track` | string | トラック名（`production`, `beta`, `alpha`, `internal` 等） |
| `releases` | Release[] | リリースの配列 |

### Release オブジェクト

| フィールド | 型 | 説明 |
|-----------|------|------|
| `name` | string | リリース名 |
| `versionCodes` | string[] | バージョンコードの配列 |
| `status` | string | ステータス（`draft`, `inProgress`, `halted`, `completed`） |
| `userFraction` | double | 段階的ロールアウトのユーザー割合 |
| `releaseNotes` | LocalizedText[] | ローカライズされたリリースノート |
| `countryTargeting` | object | 国ターゲティング |
| `inAppUpdatePriority` | integer | アプリ内更新の優先度 |

---

## Step 3: Edit の削除（edits.delete）

### エンドポイント

```
DELETE https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/edits/{editId}
```

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|------|------|
| `packageName` | string | アプリのパッケージ名 |
| `editId` | string | Edit ID |

### リクエストボディ

なし。

### レスポンス

成功時はレスポンスボディなし（HTTP 204）。

---

## ビルド番号の算出ロジック

### アルゴリズム

```
1. 全トラックを取得
2. --tracks が指定されている場合、指定トラックのみにフィルタ
3. 各トラックについて:
   a. releases が空の場合はスキップ（警告出力）
   b. 全リリースの versionCodes を収集
   c. versionCodes が存在しないリリースはスキップ
   d. 全 versionCodes を int に変換し、最大値を取得
4. 全トラックの最大値の中からさらに最大値を返す
```

### 擬似コード

```
function getLatestBuildNumber(packageName, requestedTracks):
    edit = edits.insert(packageName)
    tracks = edits.tracks.list(packageName, edit.id)
    edits.delete(packageName, edit.id)

    maxVersionCodes = {}
    for track in tracks:
        if requestedTracks is not empty AND track.name not in requestedTracks:
            continue

        allVersionCodes = []
        for release in track.releases:
            if release.versionCodes is not null:
                allVersionCodes.extend(release.versionCodes)

        if allVersionCodes is empty:
            warn("Track {track.name} has no version codes")
            continue

        maxVersionCodes[track.name] = max(allVersionCodes as integers)

    if maxVersionCodes is empty:
        error("No version codes found")

    return max(maxVersionCodes.values())
```

---

## CLI パラメータと API パラメータの対応表

| CLI パラメータ | 用途 | API パラメータ |
|---------------|------|---------------|
| `--package-name, -p` | パッケージ名の指定 | `packageName`（パスパラメータ） |
| `--tracks, -t` | フィルタ対象トラック（複数可） | — （クライアント側フィルタ） |
| `--credentials` | 認証情報 | Bearer トークンに変換して使用 |

---

## 注意事項

- `--tracks` はクライアント側のフィルタであり、API レベルでは全トラックを取得する
- Edit は読み取り専用のため、必ず `delete` で後片付けする（`commit` はしない）
- versionCode は API レスポンスでは文字列の配列として返るため、比較時に整数変換が必要
- リリースが存在しないトラックや versionCodes が null のリリースは正常にスキップする
