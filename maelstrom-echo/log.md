# 実装ログ

## 実装
[Challenge #1: Echo · Fly Docs](https://fly.io/dist-sys/1/)を見て実装していく

## 環境構築
[maelstrom/index.md at main · jepsen-io/maelstrom](https://github.com/jepsen-io/maelstrom/blob/main/doc/01-getting-ready/index.md#prerequisites)に従う

```sh
sudo apt install openjdk-17-jdk graphviz gnuplot
```

```sh
wget https://github.com/jepsen-io/maelstrom/releases/download/v0.2.3/maelstrom.tar.bz2
tar -xvf maelstrom.tar.bz2
```

Rubyは元から入っていたのでインストールしなかったが、使っていないように見えるのでインストールしなくてよい気がする

## テスト
```sh
../maelstrom/maelstrom test -w echo --bin ~/go/bin/maelstrom-echo --node-count 1 --time-limit 10
```

```
Everything looks good! ヽ(‘ー`)ノ
```
と出力されたのでOK

## その他
- `./maelstrom/`と`./store/`は.gitignoreに入れておいた
- tarballはいらなそうなので消した
