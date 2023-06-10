package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	values := struct {
		sync.RWMutex
		data map[int]struct{}
	}{
		sync.RWMutex{},
		make(map[int]struct{}),
	}

	topology := struct {
		sync.RWMutex
		data map[string][]string
	}{
		sync.RWMutex{},
		make(map[string][]string),
	}

	broadcaster := createBroadcaster(n, 10)
	defer broadcaster.drop()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		message := int(body["message"].(float64))

		values.Lock()
		_, alreadyExists := values.data[message]
		values.data[message] = struct{}{}
		values.Unlock()

		if !alreadyExists {
			topology.RLock()
			for _, neighbor := range topology.data[n.ID()] {
				broadcaster.send(bcastMessage{dest: neighbor, body: body})
			}
			topology.RUnlock()
		}

		repbody := make(map[string]any)
		repbody["type"] = "broadcast_ok"

		return n.Reply(msg, repbody)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		repbody := make(map[string]any)
		repbody["type"] = "read_ok"
		values.RLock()
		messages := make([]int, 0)
		for value := range values.data {
			messages = append(messages, value)
		}
		repbody["messages"] = messages
		values.RUnlock()

		return n.Reply(msg, repbody)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		repbody := make(map[string]any)
		repbody["type"] = "topology_ok"

		topology.Lock()
		for node, neighbors := range body["topology"].(map[string]interface{}) {
			list := make([]string, 0)
			for _, neighbor := range neighbors.([]interface{}) {
				list = append(list, neighbor.(string))
			}
			topology.data[node] = list
		}
		topology.Unlock()

		return n.Reply(msg, repbody)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}

func createBroadcaster(node *maelstrom.Node, workerNum int) *broadcaster {
	ch := make(chan bcastMessage, 1000)
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < workerNum; i++ {
		go func() {
			for {
				select {
				case msg := <-ch:
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					if _, err := node.SyncRPC(ctx, msg.dest, msg.body); err != nil {
						ch <- msg
					}
					cancel()
					break

				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return &broadcaster{
		ch: ch, cancel: cancel,
	}
}

func (bc *broadcaster) send(msg bcastMessage) {
	bc.ch <- msg
}

func (bc *broadcaster) drop() {
	bc.cancel()
}

type broadcaster struct {
	ch     chan (bcastMessage)
	cancel context.CancelFunc
}

type bcastMessage struct {
	dest string
	body map[string]any
}
