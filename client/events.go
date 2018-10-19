
package client

import (
  "fmt"
  "encoding/json"
  "strings"
  "tezos-contests.izibi.com/game/sse"
  "tezos-contests.izibi.com/game/ui"
)

type Event struct {
  Channel string `json:"channel"`
  Payload string `json:"payload"`
}

type NewBlockEvent struct {
  Hash string
  Round uint64
}

func (c *client) connectEventStream() error {
  key, err := c.remote.NewStream()
  if err != nil { return err }
  ch, err := sse.Connect(fmt.Sprintf("%s/Events/%s", c.remote.Base, key))
  if err != nil { return err }
  c.eventsKey = key
  c.eventChannel = make(chan interface{})
  go func() {
    for {
      msg := <-ch
      if msg == "" { break }
      var ev Event
      err = json.Unmarshal([]byte(msg), &ev)
      if err != nil { /* XXX report bad event */ continue }
      if ev.Channel == c.gameChannel {
        var parts = strings.Split(ev.Payload, " ")
        if len(parts) == 0 { continue }
        switch parts[0] {
        case "ping":
          ui.NoticeFmt.Println("ping")
          break
        case "block":
          c.handleBlockEvent(parts[1])
          break
        }
      }
    }
  }()
  return nil
}

func (c *client) subscribe(name string) error {
  for _, c := range c.subscriptions {
    if name == c {
      return nil
    }
  }
  if c.eventsKey != "" {
    err := c.remote.Subscribe(c.eventsKey, []string{name})
    if err != nil { return err }
  }
  c.subscriptions = append(c.subscriptions, name)
  return nil
}

func (c *client) handleBlockEvent(hash string) {
  var err error
  c.lock.Lock()
  defer c.lock.Unlock()
  /* Retrieve updated game. */
  var prevLast = c.game.LastBlock
  c.game, err = c.remote.ShowGame(c.game.Key)
  if err != nil { panic(err) /* XXX recover? */ }
  if hash == c.game.LastBlock {
    err = c.store.GetChain(prevLast, c.game.LastBlock)
    if err != nil { panic(err) /* XXX recover? */ }
    /* Save game state. */
    err = c.saveGame()
    if err != nil { panic(err) /* XXX recover? */ }
  } else {
    _, err = c.store.Get(hash)
    if err != nil { panic(err) /* XXX recover? */ }
  }
  var round uint64
  var ok bool
  round, ok = c.store.Index.GetRoundByHash(hash)
  if ok {
    c.eventChannel<- &NewBlockEvent{Hash: hash, Round: round}
  }
}
