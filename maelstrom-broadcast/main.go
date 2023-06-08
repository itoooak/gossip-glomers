package main

import (
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	values := struct {
		sync.RWMutex
		data []int
	}{
		sync.RWMutex{},
		make([]int, 0),
	}

	n.Handle("broadcast", func (msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "broadcast_ok"

		values.Lock()
		values.data = append(values.data, int(body["message"].(float64)))
		values.Unlock()

		// replyには不要な情報であるため削除
		delete(body, "message")

		return n.Reply(msg, body)
	})

	n.Handle("read", func (msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "read_ok"

		values.RLock()
		body["messages"] = values.data
		values.RUnlock()

		return n.Reply(msg, body)
	})

	n.Handle("topology", func (msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "topology_ok"

		// replyには不要な情報であるため削除
		delete(body, "topology")

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
