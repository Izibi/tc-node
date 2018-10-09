
package client

import (
  //"fmt"
  "strings"
  "tezos-contests.izibi.com/game/sse"
)

type NewBlockEvent struct {
  Hash string
  Round uint64
}

func (c *client) connectEventChannel() error {
  ch, err := sse.Connect(c.remote.Base + "/Events") // TODO: remote.Events()
  if err != nil { return err }
  c.eventChannel = make(chan interface{})
  go func() {
    for {
      e := <-ch
      // fmt.Printf("event: %v\n", e)
      if e.Type == "key" {
        c.handleKeyEvent(e.Data)
        continue
      }
      if e.Type == c.gameChannel {
        var parts = strings.Split(e.Data, " ")
        /* block hash */
        if parts[0] == "block" {
          c.handleBlockEvent(parts[1])
          continue
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

func (c *client) handleKeyEvent(key string) {
  var err error
  c.lock.Lock()
  defer c.lock.Unlock()
  c.eventsKey = key
  if len(c.subscriptions) > 0 {
    err = c.remote.Subscribe(c.eventsKey, c.subscriptions)
    if err != nil { panic(err) }
  }
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
