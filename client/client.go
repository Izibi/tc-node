
package client

import (
  "fmt"
  "io/ioutil"
  "time"

  "tezos-contests.izibi.com/game/api"
  "tezos-contests.izibi.com/game/block_store"
  "tezos-contests.izibi.com/backend/signing"
)

type Client interface {
  Start() error
  GetTimeStats() (*TimeStats, error)
  Game() *api.GameState
  NewGame(taskParams map[string]interface{}) error /* ret. key? */
  JoinGame(gameKey string) error
  SendCommands(players []PlayerConfig) error
  EndOfRound(players []PlayerConfig) error
}

type client struct {
  Task string
  remote *api.Server
  store *block_store.Store
  teamKeyPair *signing.KeyPair
  game *api.GameState
  gameChannel string
  eventsKey string
  eventChannel <-chan interface{}
  subscriptions []string
  cmdChannel chan Command
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
  err = c.connectEventChannel()
  if err != nil { return }
  err = c.loadGame()
  if err != nil { return }
  err = c.syncGame()
  if err != nil { return }
  return nil
}

func (c *client) Game() *api.GameState {
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
  var err error
  var intf string
  var impl string
  c.leaveGame()
  // fmt.Fprintf(os.Stderr, "Loading protocol\n")
  var b []byte
  b, err = ioutil.ReadFile("protocol.mli")
  if err != nil { return err }
  intf = string(b)
  b, err = ioutil.ReadFile("protocol.ml")
  if err != nil { return err }
  impl = string(b)
  if err != nil { return err }
  // fmt.Fprintf(os.Stderr, "Sending protocol\n")
  protoHash, err := c.remote.AddProtocolBlock(c.Task, intf, impl)
  if err != nil { return err }
  // fmt.Fprintf(os.Stderr, "Performing task setup\n")
  setupHash, err := c.remote.AddSetupBlock(protoHash, taskParams)
  if err != nil { return err }
  // fmt.Fprintf(os.Stderr, "Creating game\n")
  var game *api.GameState
  game, err = c.remote.NewGame(setupHash);
  c.game = game
  c.gameChannel = "games/" + game.Key
  if err != nil { return err }
  // fmt.Fprintf(os.Stderr, "Saving game state\n")
  err = c.saveGame()
  // fmt.Fprintf(os.Stderr, "Clearing store\n")
  err = c.store.Clear()
  if err != nil { return err }
  // fmt.Fprintf(os.Stderr, "Retrieving blocks\n")
  err = c.store.GetChain(game.FirstBlock, game.LastBlock)
  if err != nil { return err }
  err = c.subscribe(c.gameChannel)
  if err != nil { return err }
  return nil
}

func (c *client) JoinGame(gameKey string) error {
  var err error
  if c.game != nil {
    c.leaveGame()
  }
  // noticeFmt.Println("Retrieving game state")
  c.game, err = c.remote.ShowGame(gameKey)
  if err != nil { return err }
  c.gameChannel = "games/" + c.game.Key
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

func (c *client) SendCommands(players []PlayerConfig) (err error) {
  /* TODO: add mode where command run in parallel */
  /* Assume game is synched. */
  /* Send commands */
  for _, player := range players {
    var commands string
    commands, err = runCommand(player.CommandLine, CommandEnv{
      RoundNumber: c.game.CurrentRound,
      PlayerNumber: player.Number,
    })
    if err != nil { return }
    err = c.remote.InputCommands(
      c.game.Key, c.game.LastBlock, uint32(player.Number),
      commands)
    if err != nil { return }
  }
  return
}

func (c *client) EndOfRound(players []PlayerConfig) (err error) {
  /* Assume game is synched. */
  /* Send our commands */
  err = c.SendCommands(players)
  if err != nil { return }
  /* Close round. */
  _, err = c.remote.CloseRound(c.game.Key, c.game.LastBlock)
  if err != nil { return }
  /* Wait for end-of-turn event */
  for {
    var ev = <-c.eventChannel
    newBlockEvent := ev.(*NewBlockEvent)
    if newBlockEvent != nil {
      fmt.Printf("Round %d has ended.\n", newBlockEvent.Round)
      break
    }
  }
  /* TODO: print number of new round, and hash */
  return
}
