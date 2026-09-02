# whip-script

OBSからImageFlux Live StreamingへWHIP配信を開始するためのセットアップツールです。実行するとImageFluxの配信チャンネルを作成し、OBSの配信先を設定して配信を開始します。その後、HLSプレイリストURLと暗号鍵をlive2025-serverの内部APIへ送信します。

## 実行時の流れ

1. `config.toml` を読み込む
2. ImageFluxにマルチストリームチャンネルを作成する
3. OBS WebSocket経由でWHIP配信先を設定し、配信を開始する
4. HLSプレイリストの生成を待ち、プレイリストURLを取得する
5. HLSプレイリストとImageFlux APIから暗号鍵を取得する
6. URLと暗号鍵をlive2025-serverの内部APIへ送信する
7. 前回記録したImageFluxチャンネルがあれば削除する
8. `keys.json` に今回のチャンネルIDと暗号鍵を追記する

このプログラムは常駐監視ではなく、セットアップ完了後に終了します。OBSの配信停止はOBS側で行ってください。

## 前提条件

- OBS Studio
- OBS WebSocket（OBS 28以降では標準搭載）
- OBS側で使用可能な映像・音声ソース
- ImageFlux Live StreamingのAPIトークン
- 配信情報の登録先となるlive2025-serverと、その送信用トークン
- Dockerで実行する場合はDocker EngineとDocker Compose
- ホストで直接実行する場合はGoツールチェーン

OBSでWebSocketサーバーを有効にし、ポートとパスワードを控えてください。配信開始を伴うため、テスト時はOBSのシーンと送出内容を事前に確認してください。

## 設定ファイル

### `.env.local`

`.env.example` をコピーして作成します。

```sh
cp .env.example .env.local
```

実装が参照する環境変数は次の5つです。

| 変数 | 用途 |
| --- | --- |
| `OBS_WEBSOCKET_URL` | OBS WebSocketの接続先。ホスト実行時の例は `localhost:4455` |
| `OBS_WEBSOCKET_PASSWORD` | OBS WebSocketのパスワード |
| `STREAM_INGEST_URL` | 配信情報を登録するlive2025-serverのAPI URL |
| `STREAM_INGEST_TOKEN` | live2025-serverと共有する送信用トークン |
| `STREAM_CHANNEL_ID` | live2025-serverで使用する固定チャンネル名（`uni`、`1A`、`kaikan`、`burari`） |

`STREAM_CHANNEL_ID` はImageFluxが実行ごとに発行する配信用IDではなく、live2025-serverが認識する固定チャンネル名です。ImageFluxのトークンは環境変数ではなく、後述する `config.toml` の `imageflux.token` に設定します。

Dockerコンテナからホスト上のOBSへ接続する場合、`OBS_WEBSOCKET_URL` には通常 `host.docker.internal:4455` を指定します。`docker-compose.yaml` がLinux向けのホスト名解決を追加します。

### `config.toml`

`example.toml` をコピーして作成します。

```sh
cp example.toml config.toml
```

主な設定:

| セクション・項目 | 用途 |
| --- | --- |
| `imageflux.token` | ImageFlux APIトークン |
| `imageflux.encrypt-key-uri` | HLS暗号鍵取得先URI |
| `imageflux.event_webhook_url` | ImageFluxイベント通知先URL |
| `imageflux.hls` | 解像度、FPS、映像・音声ビットレートなどのHLS出力設定 |
| `patlite.IP` | パトライトのIP。現在の処理では未使用 |

暗号化されたHLSを使用する場合、`imageflux.encrypt-key-uri` には、配信するチャンネルに対応したBackendのHTTPS URLを指定します。たとえば `uni` では `https://<live-server>/key/uni` です。

### `keys.json`

チャンネルIDと暗号鍵の履歴を保存するファイルです。初回は空のJSON配列で作成します。

```sh
printf '[]\n' > keys.json
chmod 600 keys.json
```

`.env.local`、`config.toml`、`keys.json` はGit管理対象外です。いずれにも秘密情報が含まれるため、共有やコミットをしないでください。

## 起動方法（Docker Compose）

設定ファイルを用意し、OBSを起動してからイメージをビルドします。

```sh
docker compose build streamer
```

`--nolog` を付けると、ツール独自のログ（HLS URLを含む可能性があります）を抑止できます。処理が正常終了するとコンテナも終了します。

```sh
docker compose run --rm streamer /app/streamer --nolog
```

デバッグのためにツール独自ログが必要な場合だけ、出力先が安全であることを確認してから次を実行してください。

```sh
docker compose run --rm streamer
```

## ホストで直接実行する場合

`.env.local` はプログラム自身では自動読み込みされないため、シェルへ明示的に読み込ませます。

```sh
set -a
. ./.env.local
set +a
go run .
```

別の設定ファイルを使う場合:

```sh
go run . --config ./example.toml
```

## オプション

| オプション | 用途 |
| --- | --- |
| `--config FILE`, `-c FILE` | 読み込むTOMLファイル。既定値は `config.toml` |
| `--nolog` | ツール独自のログ出力を無効化 |
| `--nolite` | パトライト処理を無効化。現在パトライト処理自体は未実装 |

ヘルプ:

```sh
go run . --help
```

## 注意事項

- 実行するとOBSの配信設定を変更し、そのまま配信を開始します。
- HLSプレイリストが30秒以内に取得できない場合はエラー終了します。
- 前回のチャンネル削除は `keys.json` の最後の記録を基準にします。このファイルを失うと自動削除できません。
- `STREAM_INGEST_TOKEN`、ImageFluxトークン、暗号鍵は機密情報です。ログ、設定ファイル、バックアップの取り扱いに注意してください。
- Backend APIへの送信に失敗した場合はエラー終了し、後続の旧チャンネル削除と `keys.json` 更新は実行されません。

## 開発用コマンド

```sh
go test ./...
go build ./...
```
