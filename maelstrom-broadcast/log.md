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


## 3b: Multi-Node Broadcast
[Challenge #3b: Multi-Node Broadcast · Fly Docs](https://fly.io/dist-sys/3b/)

### 実装
- messageの更新が起こったノードのみbroadcastさせる
    - このようにすることで、同じ更新を要求するRPCが循環し続けるということがなくなる
        - 更新は1つのノードについて高々1回であるため
    - 過去のmessage全てをやり取りするのではなく、更新で生じた差分だけをやり取りしている
        - 全部やり取りしようとするとかえって実装が面倒になる気がする
- topology RPCで与えられた情報に従ってbroadcastの先を決める
    - topologyの更新が起こりうるので接続の情報を読み書きするときはRWLockをかける

### テスト
```sh
../maelstrom/maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast --node-count 5 --time-limit 20 --rate 10
```

```
Everything looks good! ヽ(‘ー`)ノ
```
OK

### やり残していること
- 深く考えずに`Send`を使ってbroadcastしている
    - 更新が成功しなかったときに再送する処理を入れるほうがいいのかもしれない
    - 今回のケースで仮定している状況では大丈夫そうに思えるのでスルーしてしまったが
