- [よくわかるThe Go Memory Model 用語集](#よくわかるthe-go-memory-model-用語集)
  - [逐次一貫モデル](#逐次一貫モデル)
  - [メモリー演算](#メモリー演算)
    - [メモリー演算の分類](#メモリー演算の分類)
  - [happens-before関係](#happens-before関係)
  - [concurrent(並行)](#concurrent並行)
  - [観測可能性](#観測可能性)
  - [data race](#data-race)
    - [data-race-freeとDRF-SC](#data-race-freeとdrf-sc)
# よくわかるThe Go Memory Model 用語集

- Go Conference 2023の発表「[よくわかるThe Go Memory Model](https://docs.google.com/presentation/d/1UjL5jTqreNrFpulVi6l_H5vY_Bcz9jQriL65gZs1zFM/edit?usp=sharing)」の用語集です。
- 発表中よりも少し詳しい内容を書く場合があります。

## 逐次一貫モデル

大雑把には、次のことを意味します:

> あるプログラムが「逐次一貫モデルに従う」とは、そのプログラムの実行結果が、「全ての演算を何らかの方法で一列に並べて、その通りの順序で演算を実行していった結果として説明できる」ということです。
> - ただし、その順序は、同一のgoroutineで行われる演算については、プログラムに書かれた順序を逆転させてはいけないものとします。

より正確な理解には、[How to Make a Multiprocessor Computer That Correctrly Executes Multiprocess Programs](https://www.microsoft.com/en-us/research/publication/make-multiprocessor-computer-correctly-executes-multiprocess-programs/)が役立ちます。

The Go Memory Modelでは次の表現で説明されていますが、同じ意味です。

> behave as if all the goroutines were multiplexed onto a single processor.
>
> 全てのゴルーチンが一つのプロセッサーの上に多重化されたかのように振る舞う



## メモリー演算

メモリーに読み書きする演算のことです。発表中では単に「演算」と呼びます。次のようなものは全て「メモリー演算」です。

```go
a = 1 // aに対する書き込み演算(write)
print(a) // aに対する読み込み演算(read)

var mu sync.Mutex
mu.Lock() // muに対するwrite-likeな同期演算
mu.Unlock() // muに対するread-likeな同期演算

ch := make(chan struct{})
<-ch // chに対する
ch<-struct{}{}
```

### メモリー演算の分類

メモリー演算は、"read-like","write-like"という性質で分類できます。どんなメモリー演算も、少なくとも"read-like"であるか"write-like"です。read-likeかつwrite-likeな演算もあるので、3通りに分類されます。

| 性質 | 具体例 |
| ---- | ---- |
| read-like | `print(a)`におけるaに対する読み込み, `<-ch`, `sync.(*Mutex).Lock` |
| write-like | `a = 1`, `ch<-`, `sync.(*Mutex).Unlock`|
| read-like AND write-like | `atomic.CompareAndSwap~`|


また、同期演算(synchronizing operation)とそれ以外の演算とでも分類されます。

| 同期演算かそれ以外か| 具体例 |
| ---- | ---- |
| 同期演算 |  `<-ch`, `sync.(*Mutex).Lock` |
| それ以外の演算 | `a = 1`, `print(a)`におけるaに対する読み込み|

「それ以外の演算」のことを発表中では「ふつうの演算」と呼びます。


## happens-before関係

2つの演算の間の順序関係で、`<` `>`を使って表せます。ただし、`a < b` も `b > a`も成り立たない場合があります。

- 同一goroutineの演算同士は、プログラムに書かれた順番通りにhappens-before関係が成り立ちます。
- 異なるgoroutineの演算同士は、所定の同期演算のペアになっている場合にだけhappens-before関係が成り立ちます。

## concurrent(並行)

プログラムの2つの演算がconcurrent(並行)であるとは、a happens before bが成り立たず、かつ、b happens before aも成り立たないことを言います。

## 観測可能性

あるread演算`r`が書き込み演算`w`を観測可能なのは、次のいずれかが成り立つときです。

- `w < r`であり、`w`とは別な書き込み演算`w`であって`w < w' < r`を満たすものが存在しない
- `w`と`r`が並行である

## data race

2つのメモリー演算a, bが次の条件を満たすとき、「aとbはdata raceを構成する」といいます。

- a, bは異なるgoroutineに属する
- a, bのうち少なくともどちらかが書き込み演算(write)である
- a, bの対象とするメモリー位置が重なっている(典型的には、同一変数に対する演算である)
- a, bは並行(concurrent)である

### data-race-freeとDRF-SC

あるプログラムが決してdata raceを発生させないとき、そのプログラムはdata-race-freeであるといいます。
ここで、「決して」とつけているのは、プログラムによっては実行するたびにhappens-before関係が異なる場合があるので、その発生しうるhappens-before関係のどれをとってもdata-raceが発生していないときに初めてdata-race-freeと呼ばれるからです。

プログラム言語のメモリーモデルが、「data-race-freeなプログラムに対しては逐次一貫モデルの成立を保証する」とき、そのメモリーモデルは「DRF-SCである」といいます。

Go言語をはじめとして、メモリーモデルを持つ多くの現代的プログラム言語は「DRF-SC」です。例として、C, C++, Java, JavaScript, Rustがあります。

DRF-SCなメモリーモデルは、data raceを発生させるプログラムの取り扱いによってさらに分類できます。
特に、data raceに対する振る舞いが「未定義動作」であるようなメモリーモデルは、「DRF-SC or Catch Fire」である、などと言われることがあります。
C, C++, Rustは「DRF-SC or Catch Fire」であるメモリーモデルを持ちます。
Java, JavaScript, Goは「DRF-SC or Catch Fire」ではありません。つまり、data-raceが発生した場合の振る舞いが「未定義動作」ではなく、起こりうる有限個の結果がメモリーモデルによって定められています。