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


## 3c: Fault Tolerant Broadcast
[Challenge #3c: Fault Tolerant Broadcast · Fly Docs](https://fly.io/dist-sys/3c/)

### 実装

#### 3bのコードが通るか調べる
まずはテストを走らせてみよう
分断が起こると分断をまたいだ更新が最後までかけられないため落ちるはず

```sh
Analysis invalid! (ﾉಥ益ಥ）ﾉ ┻━┻
```
予想通り落ちた

#### 検討
- broadcastが完遂できていないmessageを記録し、`broadcast`RPCが呼ばれるたびに再送するコードはどうか
    - 壊れる(実装しなくても予想できる)
    - 分断が終わった後に`broadcast`RPCが呼ばれないと完遂できていないmessageの送信は行われない
        - 分断が解消した後に`read`しか呼ばれないケースで壊れる
- `read`されたときに正しい値が読めればいいのだから、`read`されたときに更新に失敗したものがないか確認し、あれば再送するというのがよさそう
    - これは`RPC`を使う必要がある
        - 更新に成功したか失敗したかは`Send`では確認できないため
        - と思っていたが、`broadcast_ok`を受け取ったときに確認できるか
    - `read`のたびに全ノードにデータを確認するという方法もあるが、効率が悪すぎる
- GitHubで実装を探してみた
    - [teivah/gossip-glomers](https://github.com/teivah/gossip-glomers)
        - チャネルを使って送信を試みる実装になっている
        - `context`を使ったことがないので、その勉強としてもこれを参考にしてみる
        - と思ったが、このリポジトリにあったコードをテストにかけると落ちる
        - このコードでは`Send`がerrorを返してくるかで成否を判断しようとしているが、それでは不十分そう
- Fly.ioのforumを見てみる
    - [Challenge #3c: Fault Tolerant Broadcast - Questions / Help - Fly.io](https://community.fly.io/t/challenge-3c-fault-tolerant-broadcast/11289)
    - 再送を`broadcast`, `read`のたびにするという方法で解決した人がみられる
- ここまで2回ほど丸々コードを書き直している...
    - 急に難しくなっていると感じる
- `SyncRPC`でタイムアウト付き送信を行う方針で解決しそう
    - `context.WithTimeout`

#### 実装の説明
- 方針: [teivah/gossip-glomers](https://github.com/teivah/gossip-glomers)の実装に手を入れる
    - 成否の判断に問題がありそうだったので、タイムアウトを使って成否を決める実装にする
- `SyncRPC`を使う
    - `context.WithTimeout`で1秒以内に更新が成功しないとき失敗とみなしやり直す
- 失敗したRPCはキューの一番後ろに送り、成功しないものを繰り返さないようにする
    - 1度失敗したものは成功しない可能性が高いという判断
- 送信待ちのメッセージを入れたチャネルの大きさ、送信を行うgoroutineの生成数をどのように調整すれば良いのかは正直分かっていない
    - ひとまずテストが安定して通るような数値に調整した
    - 10回くらいテストを走らせたが通っているため問題はなさそうに見える
        - チャネルの大きさを10にしてみたりすると落ちる


### テスト
```sh
../maelstrom/maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast --node-count 5 --time-limit 20 --rate 10 --nemesis partition
```

```sh
Everything looks good! ヽ(‘ー`)ノ
```
OK(多分)

### やり残していること
- メッセージの送信に使うチャネルの大きさ、送信を行うgoroutineの生成数をどう調整すればいいか考える
    - ネットワークが分断される時間が長くなるほど再送しなければならないメッセージは増えていくため、チャネルの大きさはそれに合わせたものになる必要がある
    - データの読み書きが増えるほど更新に要求されるスピードは速くなっていくため、goroutineの数はそれに対応させる必要がある
    - テストを走らせ計測した結果をもとによさそうな値に調整するのも悪くはないと思うが、ちゃんと理解したうえで値を定めたいと感じている
    - まずそうだったら動的にパラメータを変える、ということも(大変ではあるが)出来そうに思える
