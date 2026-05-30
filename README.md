# gincontextleak

## 概要
gin.Contextはgoroutine-safeではないため、context.Contextインターフェース型を引数に持った関数に渡すことで競合を起こす可能性があります。
このlinterは、その関数呼び出しを検知し、`-fix`フラグを渡す場合は引数`ctx`を`ctx.Request.Context()`に修正します。

詳細
- https://github.com/haruyama480/go-gin-context-conflict

## インストール方法

```bash
go install github.com/haruyama480/gincontextleak@latest
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
