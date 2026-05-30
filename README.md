# gincontextleak

## 概要
gin.Contextはgoroutine-safeではないため、context.Contextインターフェース型を引数に持った関数に渡すことで競合を起こす可能性があります。
このlinterは、その関数呼び出しを検知し、`-fix`フラグをつけて実行する場合は引数の`*gin.Context`型の`ctx`を`ctx.Request.Context()`に修正します。

詳細
- [gin-gonic/gin#4117](https://github.com/gin-gonic/gin/issues/4117)
- https://github.com/haruyama480/go-gin-context-conflict
- https://engineering.nifty.co.jp/blog/35119

## インストール方法

```bash
go install github.com/haruyama480/gincontextleak/cmd/gincontextleak@latest
```

```bash
brew install haruyama480/tap/gincontextleak
```

## 使い方

```
gincontextleak ./...
```

```
gincontextleak -fix ./...
```

## 注意点

このlinterは、現在**関数呼び出しおよびメソッド呼び出しの引数**として `*gin.Context` が `context.Context` 型のパラメータに渡されるケースのみを検出します。

`*gin.Context` は `context.Context` インターフェースを満たすため、以下のようなケースでも暗黙的に変換されますが、これらは現時点で検出できません：

- 変数への代入（`var ctx context.Context = c` や `ctx = c`）
- 関数の戻り値として返す（`return c`）
- 構造体・スライス・マップ・チャネルなどへの格納
- クロージャや goroutine へのキャプチャ
- `interface{}` 型などを経由した間接的な受け渡し

これらのケースをすべて検出するには、データフロー解析が必要になるため、現時点ではサポート対象外です。
