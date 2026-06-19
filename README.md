# autostream-contracts

AutoStream ecosystem の shared protocol / schema repository です。

## 役割

- Control API / Observability API の OpenAPI skeleton。
- stream job、service registration、heartbeat、worker event、archive metadata、incident、remediation、notification の JSON Schema。
- permission constants と common error response。

この repository には service 実装を入れません。

## Source Of Truth

この repository は service 間 contract の source of truth です。Control Panel、Discord Bot、Encoder/Recorder、Worker、Observability が共有する field name、permission scope、error shape、event payload はここで定義し、各 service repository は自分が消費する contract だけを実装します。実装都合で各 repo に独自 schema を増やすと、runtime config、worker event、notification、archive metadata の境界が drift するため、まず `autostream-contracts` を更新してから consumer 側を合わせます。

docs には実 provider 値を置きません。schema examples では `example.com`、`<SERVICE_TOKEN>`、`<PASSWORD>` のような placeholder だけを使い、Discord guild/channel ID、Drive folder ID、YouTube stream key、webhook URL、SMTP password、OAuth refresh token の raw value は含めません。

## 変更時の責務

contract を変更するときは、どの repository が owner かを明示します。Control Panel API の request/response は `autostream-control-panel`、Discord voice / audio event は `autostream-discord-bot`、archive/upload metadata は `autostream-encoder-recorder`、overlay/caption event は `autostream-worker`、signal/incident/notification payload は `autostream-observability` が主な consumer です。

breaking change を入れる場合は、schema だけでなく migration path、compatibility period、operator docs、E2E evidence gate を同時に更新します。特に write-only secret field、runtime secret reference、service token scope、primary/standby assignment、provider proof の shape は security boundary なので、既存 field の意味を曖昧に変えないでください。

## 検証

```powershell
go test ./...
go build ./...
```

PR では、変更した schema を参照する service repo の unit test と `autostream-docs` の `npm run goal:audit` も確認します。`goal:audit` は contract 名、security regression symbol、docs hygiene の drift を拾うため、contract だけを green にして完了扱いにしません。

## Secret Policy

実 token、webhook URL、credential 付き URL は置かないでください。
例では `<SERVICE_TOKEN>`、`<PASSWORD>`、`example.com` を使います。

secret field は schema 上で write-only、masked、fingerprint、configured/missing のどれを返すかを区別します。raw secret を API response、example payload、test fixture、docs に出す必要がある設計は contract として受け入れません。PowerShell で日本語が崩れて見える場合も、source file は Node UTF-8 read で確認してから修正します。
