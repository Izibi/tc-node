
package client

import (
  "fmt"
  "encoding/json"
  "strings"
  "tezos-contests.izibi.com/tc-node/sse"
)

type Event struct {
  Channel string `json:"channel"`
  Payload string `json:"payload"`
}

type SystemEvent struct {
  Payload string
}

type EndOfGameEvent struct {
  Reason string
}

type NewBlockEvent struct {
  Hash string
}

func (cl *client) Connect() (<-chan interface{}, error) {
  if cl.eventChannel != nil {
    panic("Connect() must only be called once!")
  }
  key, err := cl.remote.NewStream()
  if err != nil { return nil, err }
  evs, err := sse.Connect(fmt.Sprintf("%s/Events/%s", cl.remote.Base, key))
  if err != nil { return nil, err }
  cl.eventsKey = key
  ech := make(chan interface{})
  go func() {
    defer evs.Close()
    for {
      msg := <-evs.C
      if msg == "" { break }
      var ev Event
      err = json.Unmarshal([]byte(msg), &ev)
      if err != nil { /* XXX report bad event */ continue }
      if ev.Channel == "system" {
        ech <- SystemEvent{Payload: ev.Payload}
        continue
      }
      if ev.Channel == cl.gameChannel {
        var parts = strings.Split(ev.Payload, " ")
        if len(parts) == 0 { continue }
        switch parts[0] {
        case "end": // ["end", reason]
          ech <- EndOfGameEvent{Reason: parts[1]}
        case "block": // ["block", hash]
          ech <- NewBlockEvent{Hash: parts[1]}
        case "ping": // ["ping" payload]
          /* Perform PONG request directly, because the worker might be busy
             doing the PING. */
          err = cl.remote.Pong(cl.game.Key, parts[1])
          if err != nil { ech<- err }
        }
      }
    }
  }()
  cl.eventChannel = ech
  return ech, nil
}

func (cl *client) subscribe(name string) error {
  err := cl.remote.Subscribe(cl.eventsKey, []string{name})
  if err != nil { return err }
  return nil
}
