# autostream-contracts

AutoStream ecosystem の shared protocol / schema repository です。

## 役割

- Control API / Observability API / Encoder Recorder API / Discord Bot API の OpenAPI skeleton。
- stream job、service registration、heartbeat、worker event、archive metadata、incident、remediation、notification の JSON Schema。
- permission constants と common error response。

この repository には service 実装を入れません。

## Source Of Truth

この repository は service 間 contract の source of truth です。Control Panel、Discord Bot、Encoder/Recorder、Worker、Observability が共有する field name、permission scope、error shape、event payload はここで定義し、各 service repository は自分が消費する contract だけを実装します。実装都合で各 repo に独自 schema を増やすと、runtime config、worker event、notification、archive metadata の境界が drift するため、まず `autostream-contracts` を更新してから consumer 側を合わせます。

docs には実 provider 値を置きません。schema examples では `example.com`、`<SERVICE_TOKEN>`、`<PASSWORD>` のような placeholder だけを使い、Discord guild/channel ID、Drive folder ID、YouTube stream key、webhook URL、SMTP password、OAuth refresh token の raw value は含めません。

## 変更時の責務

contract を変更するときは、どの repository が owner かを明示します。Control Panel API の request/response は `autostream-control-panel`、Discord voice / audio event は `autostream-discord-bot`、archive/upload metadata は `autostream-encoder-recorder`、overlay/caption event は `autostream-worker`、signal/incident/notification payload は `autostream-observability` が主な consumer です。

breaking change を入れる場合は、schema だけでなく migration path、compatibility period、operator docs、external verification flow を同時に更新します。特に write-only secret field、runtime secret reference、service token scope、primary/standby assignment、provider verification record の shape は security boundary なので、既存 field の意味を曖昧に変えないでください。

### Discord Opus ingest v2 migration

`discord-opus-ingest.schema.json` は既存の v1 producer を受ける compatibility contract として維持します。`discord-opus-ingest-v2.schema.json` は、各 packet の `job_generation` と `connection_generation` を必須にし、停止・rearm・voice reconnect 後の古い音声を fail closed で拒否する contract です。

移行順序は producer-first です。

1. `autostream-discord-bot` を先に更新し、全 packet で非ゼロの両 generation を送信させます。この期間は旧 Worker が追加 field を受け取れることを compatibility window とします。
2. Bot の status と Worker の安全な診断で両 generation が継続して送信されていることを確認します。
3. v2 を必須化した `autostream-worker` を更新します。旧 Bot と v2 Worker の組み合わせは非対応で、欠落した generation は HTTP 400、逆行した connection generation は HTTP 409 で拒否します。

運用上の compatibility floor は「v2 packet を常時送信する Discord Bot」です。Worker を先に更新しないでください。外部検証では、同一 job 内の reconnect で generation が増加し、遅延した旧 generation が新しい Deepgram connection に適用されないことを確認します。

## 検証

```powershell
go test ./...
go build ./...
```

OpenAPI YAML の構文と参照解決は、repositoryへ依存を追加せず次で確認できます。

```powershell
$bundle = Join-Path $env:TEMP 'autostream-control-api-bundle.yaml'
npx --yes @redocly/cli@2.39.0 bundle openapi/control-api.yaml --output $bundle
Remove-Item -LiteralPath $bundle -Force
npx --yes @redocly/cli@2.39.0 lint openapi/encoder-recorder-api.yaml openapi/discord-bot-api.yaml
```

PR では、変更した schema を参照する service repo の unit test と `autostream-docs` の docs consistency checks も確認します。contract 名、security regression symbol、docs hygiene の drift を拾うため、contract だけを green にして完了扱いにしません。

## Secret Policy

実 token、webhook URL、credential 付き URL は置かないでください。
例では `<SERVICE_TOKEN>`、`<PASSWORD>`、`example.com` を使います。

secret field は schema 上で write-only、masked、fingerprint、configured/missing のどれを返すかを区別します。raw secret を API response、example payload、test fixture、docs に出す必要がある設計は contract として受け入れません。PowerShell で日本語が崩れて見える場合も、source file は Node UTF-8 read で確認してから修正します。
