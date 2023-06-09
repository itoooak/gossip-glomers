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

		values.Lock()
		values.data = append(values.data, int(body["message"].(float64)))
		values.Unlock()

		repbody := make(map[string]any)
		repbody["type"] = "broadcast_ok"

		return n.Reply(msg, repbody)
	})

	n.Handle("read", func (msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		repbody := make(map[string]any)
		repbody["type"] = "read_ok"
		values.RLock()
		repbody["messages"] = values.data
		values.RUnlock()

		return n.Reply(msg, repbody)
	})

	n.Handle("topology", func (msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		repbody := make(map[string]any)
		repbody["type"] = "topology_ok"

		return n.Reply(msg, repbody)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
