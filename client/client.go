
package client

import (
  "io/ioutil"
  "time"
  "tezos-contests.izibi.com/tc-node/api"
  "tezos-contests.izibi.com/tc-node/block_store"
  "tezos-contests.izibi.com/backend/signing"
)

type Client interface {

  GetTimeStats() (*TimeStats, error)
  Connect() (<-chan interface{}, error)
  Worker() chan<- Command

  LoadGame() error
  NewGame(taskParams map[string]interface{}) error
  JoinGame(gameKey string) error
  Game() *api.GameState

}

type Notifier interface {
  Partial(msg string)
  Final(msg string)
  Error(err error)
}

type SendCommandsFeedback func(bot *BotConfig, source string, err error)

type client struct {
  task string
  remote *api.Server
  store *block_store.Store
  teamKeyPair *signing.KeyPair
  bots []BotConfig
  botRanks []uint32
  botsRegistered bool
  game *api.GameState
  gameChannel string
  eventsKey string
  eventChannel chan interface{}
  workerChannel chan<- Command
  notifier Notifier
  roundCommandsOk uint64
}

type BotConfig struct {
  Id uint32 `yaml:"id"`
  Command string `yaml:"command"`
}

type TimeStats struct {
  Local   time.Time
  Server  time.Time
  Latency time.Duration
  Delta   time.Duration
}

func New(notifier Notifier, task string, remote *api.Server, store *block_store.Store, teamKeyPair *signing.KeyPair, bots []BotConfig) Client {
  return &client{
    task: task,
    remote: remote,
    store: store,
    teamKeyPair: teamKeyPair,
    bots: bots,
    notifier: notifier,
  }
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

func (cl *client) LoadGame() error {
  var err error
  cl.notifier.Partial("Loading game state")
  err = cl.loadGame()
  if err != nil { return err }
  if cl.game == nil {
    return nil
  }
  cl.notifier.Partial("Loading store index")
  err = cl.store.Load()
  if err != nil {
    cl.notifier.Partial("Clearing corrupted store")
    cl.store.Clear()
  }
  _, err = cl.syncGame()
  if err != nil { return err }
  cl.notifier.Partial("Registering bots")
  err = cl.registerBots()
  if err != nil { return err }
  return nil
}

func (cl *client) NewGame(taskParams map[string]interface{}) error {
  var err error
  var intf string
  var impl string
  cl.notifier.Partial("Loading protocol")
  var b []byte
  b, err = ioutil.ReadFile("protocol.mli")
  if err != nil { return err }
  intf = string(b)
  b, err = ioutil.ReadFile("protocol.ml")
  if err != nil { return err }
  impl = string(b)
  if err != nil { return err }
  cl.notifier.Partial("Sending protocol")
  protoHash, err := cl.remote.AddProtocolBlock(cl.task, intf, impl)
  if err != nil { return err }
  cl.notifier.Partial("Performing task setup")
  setupHash, err := cl.remote.AddSetupBlock(protoHash, taskParams)
  if err != nil { return err }
  cl.notifier.Partial("Creating game")
  var game *api.GameState
  game, err = cl.remote.NewGame(setupHash)
  if err != nil { return err }
  cl.game = game
  cl.gameChannel = "game:" + game.Key
  cl.notifier.Partial("Saving game state")
  err = cl.saveGame()
  cl.notifier.Partial("Clearing store")
  err = cl.store.Clear()
  if err != nil { return err }
  cl.notifier.Partial("Retrieving blocks")
  err = cl.store.GetChain(game.FirstBlock, game.LastBlock)
  if err != nil { return err }
  err = cl.subscribe(cl.gameChannel)
  if err != nil { return err }
  cl.notifier.Partial("Registering bots")
  err = cl.registerBots()
  if err != nil { return err }
  return nil
}

func (cl *client) JoinGame(gameKey string) error {
  var err error
  cl.notifier.Partial("Retrieving game state")
  cl.game, err = cl.remote.ShowGame(gameKey)
  if err != nil { return err }
  cl.gameChannel = "game:" + cl.game.Key
  if err != nil { return err }
  // Subscribe to game events
  err = cl.subscribe(cl.gameChannel)
  if err != nil { return err }
  cl.notifier.Partial("Saving game state")
  err = cl.saveGame()
  if err != nil { return err }
  cl.notifier.Partial("Clearing store")
  err = cl.store.Clear()
  if err != nil { return err }
  cl.notifier.Partial("Retrieving blocks")
  err = cl.store.GetChain(cl.game.FirstBlock, cl.game.LastBlock)
  if err != nil { return err }
  cl.notifier.Partial("Registering bots")
  err = cl.registerBots()
  if err != nil { return err }
  return nil
}

func (cl *client) Game() *api.GameState {
  return cl.game
}
