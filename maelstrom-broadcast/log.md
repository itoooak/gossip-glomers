# 実装ログ

## 3a: Single-Node Broadcast
[Challenge #3a: Single-Node Broadcast · Fly Docs](https://fly.io/dist-sys/3a/)

### 実装
- 最低限の機能を実装していくことにしよう
    - 具体的には、3aではノードが1つしか存在しないため、broadcastで他のノードに値を送信する機能はつけなくてよい

- broadcastで受け取った値をノードの中で保存する際に、ロックをとる必要があるか
    - [メッセージのハンドルがgoroutineの中で行われている](https://github.com/jepsen-io/maelstrom/blob/52951329816e6df56cbdd6817d535a426aec44bf/demo/go/node.go#LL129C6-L129C19)ため、必要そう
    - ライブラリの内部を見て実装を進めていく必要が出てきた
    - `sync.Mutex`よりも`sync.RWMutex`を使うのがふさわしいように思えたのでそちらにした

### テスト
```sh
../maelstrom/maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast --node-count 1 --time-limit 20 --rate 10
```

```
Everything looks good! ヽ(‘ー`)ノ
```
通る

ノード1つの場合はそこまで考えることがない

### やり残していること
ノードを複数にした場合の対処、障害への対処は以降の節で扱うので、それは含めていない

- ~~ハンドラーを登録する際の実装があまり賢くない~~
    - Replyするときの実装がechoを使いまわしたせいで非効率になっているかもしれない
        - `delete`なんて使わずに、Reply用のbodyを別で生成する方がコードの意味的にも自然な気はする
    - 済: `repbody`という変数を用意する実装に変えた
