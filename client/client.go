
package client

import (
  "fmt"
  "os"
  "io/ioutil"
  "sync"
  "time"
  "tezos-contests.izibi.com/tc-node/api"
  "tezos-contests.izibi.com/tc-node/block_store"
  "tezos-contests.izibi.com/tc-node/ui"
  "tezos-contests.izibi.com/backend/signing"
)

type Client interface {
  Start() error
  GetTimeStats() (*TimeStats, error)
  Game() *api.GameState
  LastRoundNumber() (uint64, error)
  NewGame(taskParams map[string]interface{}) error /* ret. key? */
  JoinGame(gameKey string) error
  SyncGame() error
  SendCommands(roundNumber uint64, players []PlayerConfig, feedback SendCommandsFeedback) (bool, error)
  EndOfRound(players []PlayerConfig) error
  Events() <-chan interface{}
  Silence()
  SetNotifier(n Notifier)
}

type Notifier interface {
  Partial(msg string)
  Final(msg string)
  Error(err error)
}

type SendCommandsFeedback func(player *PlayerConfig, source string, err error)

type client struct {
  lock sync.Mutex
  Task string
  remote *api.Server
  store *block_store.Store
  teamKeyPair *signing.KeyPair
  game *api.GameState
  gameChannel string
  eventsKey string
  sendEvents bool
  eventChannel chan interface{}
  subscriptions []string
  cmdChannel chan Command
  notifier Notifier
}

type PlayerConfig struct {
  Number uint32 `yaml:"number"`
  CommandLine string `yaml:"command_line"`
}

type TimeStats struct {
  Local   time.Time
  Server  time.Time
  Latency time.Duration
  Delta   time.Duration
}

type Command interface {
  Execute() error
}

func New(task string, remote *api.Server, store *block_store.Store, teamKeyPair *signing.KeyPair) Client {
  return &client{
    Task: task,
    remote: remote,
    store: store,
    teamKeyPair: teamKeyPair,
  }
}

func (c *client) Start() (err error) {
  err = c.connectEventStream()
  if err != nil {
    ui.DangerFmt.Printf("\nYour team's public key is not recognized.\n\n")
    os.Exit(0)
  }
  err = c.loadGame()
  if err != nil {
    c.leaveGame()
    return nil
  }
  if c.game != nil {
    err = c.syncGame()
    if err != nil {
      ui.NoticeFmt.Printf("failed to synchronize game %v\n", c.game)
      c.leaveGame()
      return nil
    }
  }
  return nil
}

func (c *client) Game() *api.GameState {
  c.lock.Lock()
  defer c.lock.Unlock()
  return c.game
}

func (c *client) GetTimeStats() (*TimeStats, error) {
  var err error
  localTime := time.Now()
  var serverTime, serverTime2 time.Time
  serverTime, err = c.remote.GetTime()
  if err != nil { return nil, err }
  serverTime2, err = c.remote.GetTime()
  if err != nil { return nil, err }
  latency := serverTime2.Sub(serverTime)
  delta := serverTime.Sub(localTime) - latency
  return &TimeStats{localTime, serverTime, latency, delta}, nil
}

func (c *client) NewGame(taskParams map[string]interface{}) error {
  c.lock.Lock()
  defer c.lock.Unlock()
  var err error
  var intf string
  var impl string
  c.leaveGame()
  c.notifier.Partial("Loading protocol")
  var b []byte
  b, err = ioutil.ReadFile("protocol.mli")
  if err != nil { c.notifier.Error(err); return err }
  intf = string(b)
  b, err = ioutil.ReadFile("protocol.ml")
  if err != nil { c.notifier.Error(err); return err }
  impl = string(b)
  if err != nil { c.notifier.Error(err); return err }
  c.notifier.Partial("Sending protocol")
  protoHash, err := c.remote.AddProtocolBlock(c.Task, intf, impl)
  if err != nil { c.notifier.Error(err); return err }
  c.notifier.Partial("Performing task setup")
  setupHash, err := c.remote.AddSetupBlock(protoHash, taskParams)
  if err != nil { c.notifier.Error(err); return err }
  c.notifier.Partial("Creating game")
  var game *api.GameState
  game, err = c.remote.NewGame(setupHash);
  if err != nil { c.notifier.Error(err); return err }
  c.game = game
  c.gameChannel = "game:" + game.Key
  c.notifier.Partial("Saving game state")
  err = c.saveGame()
  c.notifier.Partial("Clearing store")
  err = c.store.Clear()
  if err != nil { c.notifier.Error(err); return err }
  c.notifier.Partial("Retrieving blocks")
  err = c.store.GetChain(game.FirstBlock, game.LastBlock)
  if err != nil { c.notifier.Error(err); return err }
  err = c.subscribe(c.gameChannel)
  if err != nil { c.notifier.Error(err); return err }
  c.notifier.Final("Game created")
  return nil
}

func (c *client) JoinGame(gameKey string) error {
  c.lock.Lock()
  defer c.lock.Unlock()
  var err error
  if c.game != nil {
    c.leaveGame()
  }
  // noticeFmt.Println("Retrieving game state")
  c.game, err = c.remote.ShowGame(gameKey)
  if err != nil { return err }
  c.gameChannel = "game:" + c.game.Key
  if err != nil { return err }
  // Subscribe to game events
  err = c.subscribe(c.gameChannel)
  if err != nil { return err }
  // noticeFmt.Println("Saving game state")
  err = c.saveGame()
  if err != nil { return err }
  // noticeFmt.Println("Clearing block store")
  err = c.store.Clear()
  if err != nil { return err }
  // noticeFmt.Println("Retrieving blocks")
  err = c.store.GetChain(c.game.FirstBlock, c.game.LastBlock)
  if err != nil { return err }
  // successFmt.Println("Success")
  return nil
}

func (c *client) SyncGame() error {
  c.lock.Lock()
  defer c.lock.Unlock()
  var err error
  c.game, err = c.remote.ShowGame(c.game.Key)
  if err != nil { return err }
  c.gameChannel = "game:" + c.game.Key
  if err != nil { return err }
  // noticeFmt.Println("Saving game state")
  err = c.saveGame()
  if err != nil { return err }
  // noticeFmt.Println("Retrieving blocks")
  err = c.store.GetChain(c.game.FirstBlock, c.game.LastBlock)
  if err != nil { return err }
  // successFmt.Println("Success")
  return nil
}

func (c *client) SendCommands(roundNumber uint64, players []PlayerConfig, feedback SendCommandsFeedback) (bool, error) {
  c.lock.Lock()
  defer c.lock.Unlock()
  /* TODO: add mode where command run in parallel */
  /* Assume game is synched. */
  /* Send commands */
  var err error
  for i := 0; i < len(players); i++ {
    var player = &players[i]
    var commands string
    commands, err = runCommand(player.CommandLine, CommandEnv{
      RoundNumber: roundNumber,
      PlayerNumber: player.Number, /* XXX should be player rank! */
    })
    if err != nil {
      feedback(player, "run", err)
      return false, err
    }
    err = c.remote.InputCommands(
      c.game.Key, c.game.LastBlock, uint32(player.Number),
      commands)
    if err != nil {
      if c.remote.LastError == "current block has changed" {
        return true, err
      }
      feedback(player, "send", err)
      return false, err
    }
    feedback(player, "ok", nil)
  }
  return false, nil
}

func (c *client) EndOfRound(players []PlayerConfig) (err error) {
  c.lock.Lock()
  defer c.lock.Unlock()
  if c.game == nil { return fmt.Errorf("no current game") }
  /* Assume game is synched, and commands have been sent. */
  /* Close round. */
  _, err = c.remote.CloseRound(c.game.Key, c.game.LastBlock)
  if err != nil { return }
  return nil
}

func (cl *client) Events() <-chan interface{} {
  cl.sendEvents = true
  return cl.eventChannel
}

func (cl *client) Silence() {
  cl.sendEvents = false
}

func (cl *client) SetNotifier(notifier Notifier) {
  cl.notifier = notifier
}

func (c *client) LastRoundNumber() (uint64, error) {
  if c.game == nil { return 0, fmt.Errorf("no current game") }
  n, ok := c.store.Index.GetRoundByHash(c.game.LastBlock)
  if !ok { return 0, fmt.Errorf("no game state on current block") }
  return n, nil
}
