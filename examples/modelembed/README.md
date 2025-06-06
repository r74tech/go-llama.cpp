# Model Embedding Example

このサンプルは、go:embedディレクティブを使用してLLMモデルファイルをGoバイナリに直接埋め込み、メモリから読み込む機能を示しています。

## 概要

go-llama.cppに追加されたメモリベースのモデル読み込み機能を使用して、モデルファイルをバイナリに埋め込むことで、単一の実行ファイルとして配布できるようになります。

## 前提条件

- Go 1.16以降（go:embedサポートのため）
- ビルド済みのlibbinding.a
- GGUFフォーマットのモデルファイル

## ビルド手順

1. まず、プロジェクトのルートディレクトリでC++ライブラリをビルドします：
```bash
cd ../..
make clean
make libbinding.a
```

2. モデルファイルを配置します：
```bash
cd examples/modelembed
# models/ディレクトリにGGUFモデルファイルを配置
# デフォルトでは tinyllama-1.1b-chat-v1.0.Q5_K_M.gguf を使用
```

3. プログラムをビルドします：
```bash
go build -o modelembed main.go
```

## 使用方法

### 埋め込みモデルを使用（メモリから読み込み）
```bash
# インタラクティブモード
./modelembed -embedded -i

# 非インタラクティブモード（単一の質問）
./modelembed -embedded -i=false -n 50 "What is the capital of France?"
```

### ファイルベースのモデルを使用（従来の方法）
```bash
# インタラクティブモード
./modelembed -m path/to/model.gguf -i

# 非インタラクティブモード
./modelembed -m path/to/model.gguf -i=false -n 50 "Hello, world!"
```

## オプション

- `-embedded`: 埋め込みモデルを使用（デフォルト: false）
- `-m`: モデルファイルのパス（デフォルト: ./models/7B/ggml-model-q4_0.bin）
- `-i`: インタラクティブモード（デフォルト: true）
- `-n`: 生成するトークン数（デフォルト: 512）
- `-t`: 計算に使用するスレッド数（デフォルト: CPUコア数）
- `-ngl`: GPUで処理するレイヤー数（デフォルト: 0）
- `-s`: 乱数シード（デフォルト: -1、ランダム）

## 実装の詳細

### モデルの埋め込み

main.goファイルの先頭で、go:embedディレクティブを使用してモデルファイルを埋め込みます：

```go
//go:embed models/tinyllama-1.1b-chat-v1.0.Q5_K_M.gguf
var embeddedModel []byte
```

### メモリからのモデル読み込み

`-embedded`フラグが指定された場合、`llama.NewFromMemory()`関数を使用してメモリからモデルを読み込みます：

```go
l, err = llama.NewFromMemory(embeddedModel,
    llama.EnableF16Memory,
    llama.SetContext(128),
    llama.EnableEmbeddings,
    llama.SetGPULayers(gpulayers))
```

## 現在の制限事項

2024年1月現在、メモリベースのモデル読み込み機能は、go-llama.cpp側では実装されていますが、llama.cpp本体での実装が必要です。そのため、現時点では以下のエラーが表示されます：

```
load_binding_model_from_memory: error: memory loading not yet fully implemented in llama.cpp
```

llama.cppライブラリに以下の機能が実装されると、完全に動作するようになります：
- `llama_load_model_from_memory`関数の実装
- GGUFファイルフォーマットのメモリ読み込み対応
- `llama_model_loader`クラスのメモリソース対応

## 利点

1. **単一実行ファイル**: モデルファイルを含む単一のバイナリとして配布可能
2. **デプロイの簡素化**: 外部ファイルへの依存がなくなる
3. **セキュリティ**: モデルファイルがバイナリ内に保護される
4. **ポータビリティ**: ファイルシステムのパスに依存しない

## 注意事項

- 埋め込むモデルのサイズによって、実行ファイルのサイズが大幅に増加します
- ビルド時にモデルファイル全体がメモリに読み込まれるため、大きなモデルの場合はビルドマシンに十分なメモリが必要です
- 実行時にもモデル全体がメモリに展開されるため、実行環境にも十分なメモリが必要です

## トラブルシューティング

### ビルドエラー: "pattern models/tinyllama-1.1b-chat-v1.0.Q5_K_M.gguf: no matching files found"
モデルファイルが指定されたパスに存在することを確認してください。

### 実行時エラー: "No embedded model found"
go:embedディレクティブで指定されたファイルがビルド時に見つからなかった場合に発生します。モデルファイルのパスを確認してください。

### メモリ不足エラー
大きなモデルを埋め込む場合、ビルドマシンと実行環境の両方に十分なメモリが必要です。より小さな量子化モデル（例：Q4_0）の使用を検討してください。