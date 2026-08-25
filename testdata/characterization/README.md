# Contracts characterization

この directory は、`pkg/contracts`、JSON Schema、4 OpenAPI の現行 semantics を、production contract を変更せず固定する test-only baseline です。`generated/` は `scripts/characterize-contracts.mjs` が管理し、手編集しません。

## 開始時 snapshot

- branch / HEAD: `main` / `0d670fa51204a950f97dc1d68c101a518d1dd2b0`（期待 base と一致、worktree clean）
- JSON Schema: 112 files、52 `$id`、136 `$ref`
- OpenAPI entrypoint: 4 files
- exported package identifier: 479（事前調査値 479 と一致）
  - 302 const、174 type、2 var、1 func、exported method 0
- exported struct-shaped type: 143
- `pkg/contracts/types.go`: 2,153 LOC
- package graph: `github.com/example/autostream-contracts/pkg/contracts` 1 package、module-local import なし、cycle なし
- Go: `go1.26.5 windows/amd64`
- Node: `v26.5.0`
- Redocly: `@redocly/cli@2.39.0`

機械可読値は `initial-state.json` にあります。OS/toolchain の将来差は contract drift と混同せず、生成物の semantic parity と分けて評価します。

## 固定する surface

- `public-api.json`: package scope の全 exported identifier、declaration kind、exact type/signature、alias/defined type、const value、var type、method receiver/signature。
- `struct-fields.json`: exported struct-shaped type、field order/type、embedded/exported、exact struct tag、JSON key、`omitempty`、`json:"-"`。
- `enum-constants.json`: exported const/enum の exact type/value。
- `zero-value-wire.json`: 143 exported struct-shaped typeを実際に `json.Marshal` した byte fingerprint、field presence、JSON shape。marshal error は 0。
- `schemas.json`: 全 Schema の path、draft、`$id`、local `$ref` target/fragment、keyword、normalized SHA-256、compile result。外部 loader は fail-closed です。
- `openapi/source-inventory.json`: 4 entrypoint と reachable local source/ref graph。HTTP(S) ref、repository 外 ref、missing local ref を拒否します。
- `openapi/normalized/*.json`: Redocly bundle を object-key sort、LF、insignificant whitespace除去、既知の volatile generator metadata除去で正規化した4 bundle。
- `openapi/fingerprints.json`: per-API normalized SHA-256 と ref-layout-independent semantic SHA-256。
- `openapi/semantic-inventory.json`: path/method/operation/security/status/body/content-type/schema/ref、operationId、inline schema、各欠落/重複検出を含む4 API inventory。
- `openapi/control-api-semantics.json`: Control API の全179 operationを1 operation単位で固定する projection。
- `openapi/lint-baseline.json`: message本文と絶対pathを除外し、severity/category/source pointer/ruleのfingerprintとして固定した現行 lint debt。

package manifest は production Go file名とcommentを入力に含めません。focused testは、同一 sourceを別名fileへ移したfixtureでも同一 manifestになることを確認します。JSON tag/value/signature/const valueは入力に含むため、これらの変更では失敗します。

Schema parser/type manifest/zero-value marshalerは Go test 内で完結します。OpenAPI は先に全 `$ref` を列挙して repository 内の実在fileへ解決し、network refを拒否してからRedoclyを呼びます。`npx` による固定CLI取得はtool bootstrapであり、contract ref解決がnetworkへfallbackすることはありません。npm cache、bundle、intermediate outputはrepository外の専用temp directoryへ置き、終了時にそのdirectoryだけを検証して削除します。

## 実行

現行 baseline を検証します。

```powershell
$env:GOTOOLCHAIN = "auto"
$env:GOMAXPROCS = "2"
node scripts/characterize-contracts.mjs verify
```

意図した contract 変更時だけ、生成後の差分を全件reviewして更新します。

```powershell
$env:GOTOOLCHAIN = "auto"
$env:GOMAXPROCS = "2"
node scripts/characterize-contracts.mjs update
git diff -- testdata/characterization/generated
```

この Task のような production contract 非変更作業では、baseline update は現 HEAD の初回採取にだけ使います。lint debtを0へ書き換えたり、sourceをbaselineへ合わせたりしません。

## move-only before/after verifier

`types.go` のsame-package物理分割またはOpenAPIのlocal `$ref` layout変更を別Taskで行う場合、変更前後をrepository外へ採取します。

```powershell
$characterizationBefore = Join-Path ([System.IO.Path]::GetTempPath()) ("autostream-contracts-before-" + [guid]::NewGuid().ToString("N"))
$characterizationAfter = Join-Path ([System.IO.Path]::GetTempPath()) ("autostream-contracts-after-" + [guid]::NewGuid().ToString("N"))

node scripts/characterize-contracts.mjs snapshot --output $characterizationBefore
# authorized move-only change
node scripts/characterize-contracts.mjs snapshot --output $characterizationAfter
node scripts/characterize-contracts.mjs compare --before $characterizationBefore --after $characterizationAfter
```

`compare` は public API、field/tag、enum、zero wire、Schema semantics、ref-layout-independent bundle semantics、operation semantics を比較します。source file path、lint source location、raw ref/component name、layout-sensitive normalized bundle hashは比較対象外です。一方、通常の `verify` はそれらも含むstrict baselineなので、source/ref/lint layout driftは明示的なbaseline reviewなしに通りません。

## 現行 OpenAPI inventory

| API | paths | operations | operationId | schemas | security schemes | request bodies | unauthenticated | classified public | inline schemas | no 2xx | no 4xx | lint error/warning |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Control | 135 | 179 | 14 | 129 | 2 | 55 | 11 | 6 | 20 | 4 | 114 | 156 / 410 |
| Discord Bot | 3 | 3 | 3 | 5 | 1 | 2 | 1 | 0 | 1 | 0 | 1 | 0 / 3 |
| Encoder/Recorder | 4 | 4 | 4 | 7 | 1 | 2 | 1 | 0 | 4 | 0 | 1 | 0 / 11 |
| Observability | 21 | 25 | 2 | 11 | 1 | 4 | 3 | 0 | 4 | 0 | 19 | 23 / 46 |

すべてbundle成功、unresolved ref 0、method/path重複0、duplicate operationId 0、security未定義0、responseなし0です。`no 2xx`、`no 4xx`、operationId不足、lint error/warningは今回修正しない現行debtであり、category/path/rule fingerprintとしてbaseline化しています。

`security: []` を無条件にpublicとは扱いません。Controlのlogin/OAuth provider・start・callback、public app settings、conditional first-adminだけを現在のintended/conditional public inventoryに入れ、health probe、loopback updater、capability URL、grant-token endpoint、Observability internal statusは別classificationです。新しいunauthenticated operationは明示classificationがない限り生成を拒否します。

## authority と consumer 境界

分類: **RAW_SOURCE_AUTHORITY**

- repository README はこのrepositoryをservice間contractのsource of truthとしています。
- release workflowは `openapi/`、`schemas/`、`README.md` をそのままarchiveへcopyします。
- release assetへRedocly bundleを入れる処理はありません。bundleは検証用temp outputです。
- したがって現時点で `BUNDLE_AUTHORITY` または `BOTH` とは確認できません。raw treeに相対 `$ref` が残るため、外部consumerがpath/layoutへ依存する可能性を否定できません。

既存 `.github/workflows/ci.yml` の Redocly job は固定版2.39.0で Discord Bot OpenAPI のlint/bundleと `job_generation` 境界を検証しますが、Control、Encoder/Recorder、Observabilityのbundle/lintは実行していません。この characterization は4 APIを対象にしますが、本Taskではworkflowを変更しません。

5 consumerの読み取り専用scanは `consumer-usage.json` に記録しています。contracts Go moduleの直接importは0ですが、same-name DTO候補と共通JSON tagは多数あります。Control Panelのinternal task docsにはrelative OpenAPI/Schema path記載があります。runtime/buildのSchema/OpenAPI path依存、generated client、contracts release artifact path依存は5 checkoutでは見つかりませんでした。ただしworkspace外consumer不存在の証明ではありません。

このcharacterizationは Schema validator責務、Go decoder numeric range、JSON wire requiredness、OpenAPI documentationを別artifactで固定します。`job_generation` は Schema required/integer/minimum 1/maximumなし、Go `uint64`/`omitempty`なし、MaxUint64受理/overflow拒否を既存testとbundle characterizationの両方で維持します。

OpenAPI source split、`types.go` split、authority変更、production refactor、workflow変更、release変更はこのdirectoryの作成範囲外です。
