# よくわかるThe Go Memory Model

Go Conference 2023 発表「よくわかるThe Go Memory Model」の文字起こしです。

- 元スライド: [よくわかるThe Go Memory Model](https://docs.google.com/presentation/d/1UjL5jTqreNrFpulVi6l_H5vY_Bcz9jQriL65gZs1zFM/edit?usp=sharing)
- 用語の詳細については[用語集](./glossary.md)も参照してください。

---

## 目次

1. [The Go Memory Modelとは](#the-go-memory-modelとは)
2. [なぜメモリーモデルが必要か](#なぜメモリーモデルが必要か)
3. [メモリー演算とは](#メモリー演算とは)
4. [happens-before関係](#happens-before関係)
5. [逐次一貫モデル](#逐次一貫モデル)
6. [concurrent（並行）](#concurrent並行)
7. [観測可能性](#観測可能性)
8. [data race](#data-race)
9. [DRF-SC](#drf-sc)
10. [Goにおける同期演算の保証](#goにおける同期演算の保証)
11. [間違った同期の例](#間違った同期の例)
12. [まとめ](#まとめ)

---

## The Go Memory Modelとは

**The Go Memory Model**（Goメモリーモデル）とは、次のことを規定するドキュメントです。

> あるgoroutineの中で行われた変数への書き込みが、別のgoroutineから読み取れることが**保証される**条件

公式ドキュメント: https://go.dev/ref/mem

### なぜ難しいのか

並行プログラムでは、直感に反する動作が起きることがあります。現代のコンピューターは、パフォーマンスのために命令の順序を変えることがあります（コンパイラーの最適化やCPUの命令の並列実行・リオーダーなど）。

この発表では、The Go Memory Modelを正確に理解し、安全な並行プログラムを書くための知識を提供します。

---

## なぜメモリーモデルが必要か

### 動機となる例

次のプログラムを考えてみます（[messagepassing/main.go](./messagepassing/main.go)）：

```go
var a, b int64
var wg sync.WaitGroup

func f() {
    defer wg.Done()
    a = 1  // ①
    b = 1  // ②
}

func g() {
    defer wg.Done()
    r1 := b  // ③
    r2 := a  // ④
    if r1 == 1 && r2 == 0 { // これは発生しうるか？
        panic("Answer: Yes")
    }
}

func exec() {
    a = 0
    b = 0
    wg.Add(2)
    defer wg.Wait()

    go f()
    go g()
}

func main() {
    for {
        exec()
    }
}
```

**問題**: goroutine `g` が `r1 == 1 && r2 == 0` という結果を観測することがありうるか？

直感的には、`f` が `a = 1` を実行してから `b = 1` を実行しているので、`b == 1` が見えているなら `a == 1` も見えているはずに思えます。

**実際には: Yes、これは発生しうる。**

このプログラムを実行すると、実際にpanicが発生します。

### なぜ発生するのか

CPUやコンパイラーは、同一goroutine内から見て正しいのであれば、演算の順序を入れ替えることがあります。

- `f` の中では `a = 1` の前に `b = 1` が実行されることがある
- `g` の中では `r2 := a` の前に `r1 := b` が実行される順番が変わることがある

このような再順序付けは、The Go Memory Modelが許可しているものです。

### 解決策: atomic操作を使う

同じプログラムをatomicを使って書き直すと（[atomics/main.go](./atomics/main.go)）：

```go
var a, b atomic.Int64
var wg sync.WaitGroup

func f() {
    defer wg.Done()
    a.Store(1)  // ①（atomic書き込み）
    b.Store(1)  // ②（atomic書き込み）
}

func g() {
    defer wg.Done()
    r1 := b.Load()  // ③（atomic読み込み）
    r2 := a.Load()  // ④（atomic読み込み）
    if r1 == 1 && r2 == 0 {
        panic("Answer: Yes")
    }
}
```

atomic操作は**同期演算**（synchronizing operation）であり、happens-before関係を確立します。これにより、`r1 == 1 && r2 == 0` という結果は発生しなくなります。

なぜ発生しなくなるかは、次のセクション以降で詳しく説明します。

---

## メモリー演算とは

### 定義

**メモリー演算**（memory operation）とは、メモリーに対して読み書きする演算のことです。次のようなものが含まれます。

```go
a = 1              // a に対する書き込み演算（write）
print(a)           // a に対する読み込み演算（read）

var mu sync.Mutex
mu.Lock()          // mu に対する read-like な同期演算
mu.Unlock()        // mu に対する write-like な同期演算

ch := make(chan struct{})
<-ch               // ch に対する read-like な同期演算
ch <- struct{}{}   // ch に対する write-like な同期演算
```

### メモリー演算の分類: read-like と write-like

メモリー演算は **read-like** と **write-like** という性質で分類できます。

| 性質 | 具体例 |
| ---- | ---- |
| read-like | `print(a)` における a の読み込み、`<-ch`、`sync.(*Mutex).Lock` |
| write-like | `a = 1`、`ch <-`、`sync.(*Mutex).Unlock` |
| read-like かつ write-like | `atomic.CompareAndSwap~` |

### メモリー演算の分類: 同期演算とふつうの演算

また、**同期演算**（synchronizing operation）かどうかでも分類されます。

| 分類 | 具体例 |
| ---- | ---- |
| 同期演算 | `<-ch`、`sync.(*Mutex).Lock`、`atomic.Load`、`atomic.Store` など |
| ふつうの演算 | `a = 1`、`print(a)` における a の読み込み |

**同期演算だけがhappens-before関係を確立できます。**

---

## happens-before関係

### 定義

**happens-before関係**とは、2つのメモリー演算の間の順序関係です。`a < b` と書いて「a は b より前に起きた（a happens before b）」を表します。

ただし、`a < b` も `b < a` も成り立たない場合があります（これを**concurrent**と言います）。

### 同一goroutineの場合

同一goroutine内では、プログラムに書かれた順番通りにhappens-before関係が成り立ちます。

```go
// 同一goroutine内
a = 1    // ①
b = 2    // ②
print(a) // ③

// ① happens-before ②
// ② happens-before ③
// ① happens-before ③
```

### 異なるgoroutineの場合

異なるgoroutine間では、所定の**同期演算のペア**になっている場合にだけhappens-before関係が成り立ちます。

```go
// goroutine 1             // goroutine 2
a = "hello"               
ch <- struct{}{}          <-ch
                          print(a) // "hello" と出力されることが保証される
```

この例では、チャネル送信とチャネル受信が同期演算のペアを形成し、happens-before関係を確立します。

### happens-before関係の推移性

happens-before関係は推移的（transitive）です。つまり、`a < b` かつ `b < c` ならば `a < c` が成り立ちます。

---

## 逐次一貫モデル

### 定義

大雑把には、次のことを意味します：

> あるプログラムが「逐次一貫モデルに従う」とは、そのプログラムの実行結果が、「全ての演算を何らかの方法で一列に並べて、その通りの順序で演算を実行していった結果として説明できる」ということです。
> - ただし、その順序は、同一のgoroutineで行われる演算については、プログラムに書かれた順序を逆転させてはいけないものとします。

The Go Memory Modelでは次の表現で説明されています：

> behave as if all the goroutines were multiplexed onto a single processor.
>
> 全てのゴルーチンが一つのプロセッサーの上に多重化されたかのように振る舞う

---

## concurrent（並行）

### 定義

2つのメモリー演算 `a`、`b` について：

- `a happens-before b` が成り立たない
- かつ、`b happens-before a` も成り立たない

このとき、「`a` と `b` は **concurrent（並行）** である」と言います。

### 図解

```
goroutine 1: ----[a=1]----[b=1]------------>
                                  ↑ 
goroutine 2: --------[r1:=b]--[r2:=a]----->
```

同期演算がない場合、異なるgoroutine間の演算はconcurrentです。

---

## 観測可能性

### 定義

あるread演算 `r` が write演算 `w` を**観測可能**（visible）なのは、次のいずれかが成り立つときです。

1. `w happens-before r` であり、かつ `w` とは別な write演算 `w'` であって `w happens-before w' happens-before r` を満たすものが存在しない
2. `w` と `r` が concurrent（並行）である

### ふつうの演算の場合

**ふつうの（非同期の）read演算** `r` については、`W(r)` として選ばれる write演算は、`r` から観測可能な write演算のいずれかです。

これが問題になるのは、concurrent な write演算が複数ある場合です。どの値が読まれるかは**不定**です。

### 同期演算の場合

**同期演算** `r` については、`W(r)` として選ばれる write演算は、何らかの全体的な順序によって説明できる必要があります。

---

## data race

### 定義

2つのメモリー演算 `a`、`b` が次の条件を全て満たすとき、「`a` と `b` は **data race** を構成する」と言います。

1. `a`、`b` は異なるgoroutineに属する
2. `a`、`b` のうち少なくともどちらかが write演算である
3. `a`、`b` の対象とするメモリー位置が重なっている（典型的には、同一変数への演算）
4. `a`、`b` は concurrent（並行）である

### 例

```go
var a = 0

func main() {
    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        a = 1  // ① goroutine 1 が a に書き込む
        wg.Done()
    }()
    go func() {
        a = 2  // ② goroutine 2 が a に書き込む
        print(a)
        wg.Done()
    }()
    wg.Wait()
}
```

（[happensbefore/sample1/main.go](./happensbefore/sample1/main.go) を参照）

① と ② は concurrent であり、両方が write演算なので、これはdata raceです。

### data raceの問題点

data raceがあるプログラムは**未定義の動作**に近い振る舞いをする可能性があります（Goの場合は後述の通り限定的ですが）。`go build -race` でdata raceを検出できます。

---

## DRF-SC

### data-race-free（DRF）

あるプログラムが決してdata raceを発生させないとき、そのプログラムは**data-race-free（DRF）** であると言います。

### DRF-SC

プログラム言語のメモリーモデルが「data-race-freeなプログラムに対しては逐次一貫モデルの成立を保証する」とき、そのメモリーモデルは **DRF-SC** であると言います。

**Go言語のメモリーモデルはDRF-SCです。**

つまり：

> data raceが発生しないプログラムは、全てのgoroutineが一つのプロセッサー上で多重化されたかのように振る舞います。

### GoはDRF-SC or Catch Fireではない

DRF-SCなメモリーモデルは、data raceが発生した場合の振る舞いによってさらに分類できます。

- **DRF-SC or Catch Fire**: data raceが発生した場合は**未定義動作**（C/C++、Rust）
- **DRF-SC（Catch Fireではない）**: data raceが発生した場合でも起こりうる結果が有限個に限定される（Java、JavaScript、**Go**）

Goのメモリーモデルでは、data raceが発生した場合でも：
- 単語サイズ以下のメモリー位置の read は、実際にそのメモリー位置に書き込まれた値（過去または並行して）を観測しなければならない
- "out of thin air"（何もないところからの値の生成）は許可されない

---

## Goにおける同期演算の保証

The Go Memory Modelは、特定の演算がhappens-before関係を確立することを保証しています。

### 初期化（Initialization）

- `p` が `q` をimportしているとき、`q` の全 `init` 関数の完了は `p` の全 `init` 関数の開始より前にhappens-before
- 全ての `init` 関数の完了は `main.main` の開始より前にhappens-before

### goroutineの生成

`go` 文による goroutine の起動は、その goroutine の実行開始より前にhappens-before。

```go
var a string

func f() {
    print(a) // "hello, world" が出力されることが保証される
}

func hello() {
    a = "hello, world" // go f() より前に実行される
    go f()
}
```

### goroutineの終了

**goroutineの終了は、プログラム中のどんなイベントとも synchronized before の関係を確立しません。**

```go
var a string

func hello() {
    go func() { a = "hello" }() // この終了は保証されない
    print(a)                     // 空文字列かもしれない
}
```

### チャネル通信

チャネルはgoroutine間の主要な同期手段です。

**全てのチャネル:**

| 演算 | 保証 |
| ---- | ---- |
| 送信（`ch <-`） | 対応する受信（`<-ch`）の**完了**より前にsynchronized before |
| `close(ch)` | チャネルが閉じていることを示すゼロ値の受信より前にsynchronized before |

**バッファなしチャネル（unbuffered channel）のみ:**

| 演算 | 保証 |
| ---- | ---- |
| 受信（`<-ch`） | 対応する送信（`ch <-`）の**完了**より前にsynchronized before |

バッファなしチャネルでは送信と受信が「待ち合わせ」を行うため、送信の前に書かれたデータは受信の後に読めることが保証され、逆も然りです。

**バッファありチャネル（capacity = C のチャネル）:**

k番目の受信（`<-ch`）は (k+C)番目の送信（`ch <-`）の完了より前にsynchronized before。

これはバッファありチャネルをカウンティングセマフォとして使えることを意味します。

```go
var c = make(chan int, 10)
var a string

func f() {
    a = "hello, world"
    c <- 0 // ← 送信。a = "hello, world" より後
}

func main() {
    go f()
    <-c    // ← 受信。上の送信が完了してから戻る
    print(a) // "hello, world" が保証される
}
```

### Mutex（排他ロック）

`sync.Mutex` および `sync.RWMutex` の保証：

n回目の `Unlock()` は (n+1)回目の `Lock()` の完了より前にhappens-before。

```go
var l sync.Mutex
var a string

func f() {
    a = "hello, world"
    l.Unlock() // ← 2回目のLockより前にhappens-before
}

func main() {
    l.Lock()
    go f()
    l.Lock()   // ← f()のUnlockより後に完了する
    print(a)   // "hello, world" が保証される
}
```

### atomic操作

`sync/atomic` パッケージのatomic操作は同期演算です。

あるatomic操作 `A` の効果が atomic操作 `B` から観測された場合、`A` は `B` より前にsynchronized before。

**全ての atomic操作はプログラム全体で逐次一貫的に実行されます。**

これが最初に示した `messagepassing` の例で、atomicを使うことで `r1 == 1 && r2 == 0` が発生しなくなる理由です：

```
goroutine f:  a.Store(1) → b.Store(1)
goroutine g:  r1 := b.Load() → r2 := a.Load()
```

`b.Load()` が `b.Store(1)` の結果（1）を観測したとき、atomic操作の順序により `a.Store(1)` は `b.Store(1)` より前にhappens-before、したがって `a.Load()` は必ず 1 を返します。

### sync.Once

`once.Do(f)` は `f` を一度だけ実行します。

`f()` の完了は `once.Do(f)` の全ての返り値より前にhappens-before。

---

## 間違った同期の例

### ダブルチェックロック（Double-checked locking）

```go
var a string
var done bool
var once sync.Once

func setup() {
    a = "hello, world"
    done = true
}

func doprint() {
    if !done {
        once.Do(setup)
    }
    print(a) // ← "hello, world" が観測される保証はない！
}
```

`done == true` が観測されたとしても、`a == "hello, world"` が観測される保証はありません。`done` と `a` の間にhappens-before関係がないからです。

### ビジーウェイト（Busy waiting）

```go
var a string
var done bool

func setup() {
    a = "hello, world"
    done = true
}

func main() {
    go setup()
    for !done { // ← このループが終わらない可能性がある
    }
    print(a) // ← "hello, world" が観測される保証はない！
}
```

`done` への書き込みと `done` の読み込みの間に同期演算がないため：
1. ループが終わらない可能性がある
2. ループが終わっても `a` の値が観測される保証はない

**解決策は常に同じ: 明示的な同期を使う（チャネル、Mutex、atomic）**

---

## まとめ

### The Go Memory Modelのポイント

1. **メモリー演算の分類**: read-like / write-like、同期演算 / ふつうの演算
2. **happens-before関係**: 演算間の順序関係。同期演算のペアによってのみ確立される
3. **concurrent（並行）**: happens-before関係がない2つの演算
4. **data race**: concurrent で少なくとも一方が write の演算ペア
5. **DRF-SC**: data-race-freeなプログラムは逐次一貫的に動作する（Goはこれを保証する）

### 実践的な指針

> If you must read the rest of this document to understand the behavior of your program, you are being too clever.
>
> Don't be clever.
>
> （プログラムの動作を理解するためにこの文書を読む必要があるなら、あなたは賢すぎます。賢くなりすぎないようにしましょう。）
>
> — [The Go Memory Model](https://go.dev/ref/mem)

- 複数のgoroutineからアクセスされるデータは、チャネル操作またはsync/sync.atomicパッケージの同期プリミティブで保護する
- `go build -race` でdata raceを検出する
- 「同期しなくても大丈夫だろう」という直感を信じない

### Goのメモリーモデルの位置づけ

| 言語 | DRF-SC | data raceの扱い |
| ---- | ---- | ---- |
| C/C++ | ✓ | 未定義動作（"Catch Fire"） |
| Rust | ✓ | 未定義動作（"Catch Fire"） |
| Java | ✓ | 限定的な結果 |
| JavaScript | ✓ | 限定的な結果 |
| **Go** | ✓ | **限定的な結果**（単語サイズ以下のメモリーは実際に書き込まれた値のいずれか） |

---

*この文書は Go Conference 2023 の発表スライドをもとに作成しました。*
*元スライド: https://docs.google.com/presentation/d/1UjL5jTqreNrFpulVi6l_H5vY_Bcz9jQriL65gZs1zFM/edit?usp=sharing*
