
package client

import (
  "bufio"
  "errors"
  "fmt"
  "io"
  "os"
)

func (cl *client) Worker() (chan<- Command, chan<- Command) {
  if cl.workerRunning {
    panic("Worker() must only be called once!")
  }
  var wch = make(chan Command, 1)
  var ich = NewIdleChannel()
  go func() {
    for {
      var cmd Command
      select {
      case cmd = <-wch:
        fmt.Printf("Processing command")
      case cmd = <-ich.Out():
        fmt.Printf("Processing idle command")
      }
      err := cmd.run(cl)
      if err != nil {
        // TODO: possibly send an event, so this is displayed in the interactive loop?
        cl.notifier.Error(err)
      }
    }
  }()
  cl.workerRunning = true
  return wch, ich.In()
}

type Command struct {
  run func (cl *client) error
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
    if len(cl.bots) == 0 {
      cl.notifier.Final("no bots configured!")
      return nil
    }
    var err error
    var currentRound uint64
    currentRound, err = cl.lastRoundNumber()
    if err != nil { return err }
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

type BotFeedback struct {
  Status string
  Bot *BotConfig
  Round uint64
  Rank uint32
  Err error
}

func (cl *client) sendCommands(currentRound uint64) error {
  var err error
  cl.notifier.Final(fmt.Sprintf("Sending commands for round %d", currentRound))
  var retry bool
  for {
    retry, err = cl.trySendCommands(currentRound)
    if err == nil {
      cl.roundCommandsOk = currentRound
    }
    if !retry {
      return err
    }
    currentRound, err = cl.syncGame()
    if err != nil {
      return err
    }
  }
}

func (cl *client) trySendCommands(roundNumber uint64) (bool, error) {
  var err error
  var log *os.File
  var lastError error

  log, err = os.OpenFile("commands.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
    0644)
  if err != nil {
    cl.notifier.Error(errors.New("failed to write commands.log"))
    log = nil
  }
  if log != nil {
    defer log.Close()
    log.WriteString(fmt.Sprintf("Round: %d\n", roundNumber))
    log.WriteString(fmt.Sprintf("NbCycles: %d\n", cl.game.NbCyclesPerRound))
  }

  for i := range cl.bots {
    bot := &cl.bots[i]
    if i >= len(cl.botRanks) {
      /* Some bots are not playing because the game is full. */
      break
    }
    rank := cl.botRanks[i]

    fmt.Printf("--- START bot id %d --- player %d --- round %d ---\n",
      bot.Id, rank, roundNumber)
    if log != nil {
      log.WriteString(fmt.Sprintf("\n--- Player %d BotId %d ---\n", rank, bot.Id))
    }

    var commands string
    commands, err = runCommand(bot.Command, CommandEnv{
      BotId: bot.Id,
      RoundNumber: roundNumber,
      PlayerNumber: rank,
      NbCycles: cl.game.NbCyclesPerRound,
    })
    if err != nil {
      lastError = err
      if log != nil {
        log.WriteString(fmt.Sprintf("\nError running bot: %v\n", err))
      }
      cl.notifier.Error(fmt.Errorf("Bot id %d error -- see commands.log", bot.Id))
      continue
    }
    if log != nil {
      log.WriteString(commands)
    }

    err = cl.remote.InputCommands(cl.game.Key, cl.game.LastBlock, bot.Id, commands)
    if err != nil {
      if cl.remote.LastError == "current block has changed" {
        if log != nil {
          log.WriteString("\nCommands were sent after end of block, and ignored.\n")
        }
        cl.notifier.Error(fmt.Errorf("Bot id %d was too slow", bot.Id))
        return true, err // retry
      }
      if log != nil {
        log.WriteString(fmt.Sprintf("\nError sending commands: %v\n", err))
      }
      cl.notifier.Error(err)
      return false, err
    }

    fmt.Printf("--- READY bot id %d --- player %d --- round %d ---\n",
      bot.Id, rank, roundNumber)
  }

  return false, lastError
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
