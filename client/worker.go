
package client

import (
  "io"
  "bufio"
  "fmt"
)

func (cl *client) Worker() chan<- Command {
  if cl.workerChannel != nil {
    panic("Worker() must only be called once!")
  }
  var wch = make(chan Command, 1)
  go func() {
    var busy bool
    for {
      cmd := <-wch
      if busy {
        if cmd.isSignal {
          continue
        } else {
          panic("can't")
        }
      }
      go func(cmd Command) {
        busy = true
        err := cmd.run(cl)
        if err != nil {
          // TODO: possibly send an event, so this is displayed in the interactive loop?
          cl.notifier.Error(err)
        }
        busy = false
      }(cmd)
    }
  }()
  cl.workerChannel = wch
  return wch
}

type Command struct {
  run func (cl *client) error
  isSignal bool
}

func (c Command) Signal() Command {
  return Command{c.run, true}
}

func Ping() Command {
  run := func(cl *client) error {
    cl.notifier.Partial("Pinging all nodes playing on this game")
    var err error
    var rc io.ReadCloser
    rc, err = cl.remote.Ping(cl.game.Key)
    if err != nil { return err }
    br := bufio.NewReader(rc)
    defer rc.Close()
    for {
      var bs []byte
      bs, err = br.ReadBytes('\n')
      if err != nil { return err }
      line := string(bs[0:len(bs)-1])
      switch line {
      case "OK":
        cl.notifier.Final("All nodes are responsive")
        return nil
      case "timeout":
        cl.notifier.Final("Some nodes are unresponsive")
        return nil
      }
    }
    return nil
  }
  return Command{run: run}
}

func AlwaysSendCommands() Command {
  run := func(cl *client) error {
    fmt.Println("AlwaysSendCommands")
    if len(cl.players) == 0 {
      cl.notifier.Final("no players configured!")
      return nil
    }
    var err error
    var currentRound uint64
    currentRound, err = cl.lastRoundNumber()
    if err != nil { return err }
    fmt.Println("AlwaysSendCommands ->")
    return cl.sendCommands(currentRound)
  }
  return Command{run: run}
}

func Sync() Command {
  run := func(cl *client) error {
    _, err := cl.syncGame()
    return err
  }
  return Command{run: run}
}

func SyncThenSendCommands() Command {
  run := func(cl *client) error {
    var err error
    var currentRound uint64
    currentRound, err = cl.syncGame()
    if err != nil { return err }
    if cl.roundCommandsOk != currentRound {
      err = cl.sendCommands(currentRound)
      if err != nil { return err }
    }
    return nil
  }
  return Command{run: run}
}

type LocalPlayerFeedback struct {
  Player *PlayerConfig
  Step string
  Err error
}

func (cl *client) sendCommands(currentRound uint64) error {
  var err error
  cl.notifier.Final(fmt.Sprintf("Sending commands for round %d", currentRound))
  var retry bool
  for {
    retry, err = cl.trySendCommands(currentRound)
    if !retry {
      return err
    }
    currentRound, err = cl.syncGame()
    if err != nil {
      return err
    }
  }
  cl.roundCommandsOk = currentRound
  return nil
}

func (cl *client) trySendCommands(roundNumber uint64) (bool, error) {
  var err error
  for i, player := range(cl.players) {
    rank := cl.playerRanks[i]
    var commands string
    cl.eventChannel <- LocalPlayerFeedback{&player, "begin", nil}
    commands, err = runCommand(player.CommandLine, CommandEnv{
      RoundNumber: roundNumber,
      PlayerNumber: rank, // was player.Number
    })
    if err != nil {
      cl.eventChannel <- LocalPlayerFeedback{&player, "run", err}
      return false, err
    }
    err = cl.remote.InputCommands(
      cl.game.Key, cl.game.LastBlock, player.Number,
      commands)
    if err != nil {
      if cl.remote.LastError == "current block has changed" {
        return true, err
      }
      cl.eventChannel <- LocalPlayerFeedback{&player, "send", err}
      return false, err
    }
    cl.eventChannel <- LocalPlayerFeedback{&player, "ready", nil}
  }
  return false, nil
}

func EndOfRound() Command {
  run := func(cl *client) error {
    var err error
    var currentRound uint64
    currentRound, err = cl.lastRoundNumber()
    if err != nil { return err }
    cl.notifier.Partial(fmt.Sprintf("Closing round %d", currentRound))
    _, err = cl.remote.CloseRound(cl.game.Key, cl.game.LastBlock)
    if err != nil { return err }
    cl.notifier.Final(fmt.Sprintf("Round %d is closed", currentRound))
    return nil
  }
  return Command{run: run}
}

/*


func (cl *client) endOfRound(players []PlayerConfig) (err error) {
  return nil
}

func (cl *client) handleBlockEvent(hash string) {
  var err error
  // Retrieve updated game.
  var prevLast = cl.game.LastBlock
  cl.game, err = cl.remote.ShowGame(cl.game.Key)
  if err != nil { panic(err) XXX recover? }
  if hash == cl.game.LastBlock {
    err = cl.store.GetChain(prevLast, cl.game.LastBlock)
    if err != nil { panic(err) XXX recover? }
    // Save game state.
    err = cl.saveGame()
    if err != nil { panic(err) XXX recover? }
  } else {
    _, err = cl.store.Get(hash)
    if err != nil { panic(err) XXX recover? }
  }
  var round uint64
  var ok bool
  round, ok = cl.store.Index.GetRoundByHash(hash)
  fmt.Printf("round %d %v\n", round, ok)
}
*/
