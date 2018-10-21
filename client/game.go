
package client

import (

  "os"
  "io/ioutil"
  "encoding/json"
  "bytes"
  "tezos-contests.izibi.com/tc-node/api"
)

func (c *client) loadGame() error {
  var err error
  var b []byte
  filepath := "game.json"
  _, err = os.Stat(filepath)
  if os.IsNotExist(err) {
    c.game = nil
    c.gameChannel = ""
    return nil
  }
  b, err = ioutil.ReadFile(filepath)
  if err != nil { return err }
  game := new(api.GameState)
  err = json.NewDecoder(bytes.NewBuffer(b)).Decode(game)
  if err != nil { return err }
  c.game = game
  c.gameChannel = "game:" + game.Key
  err = c.subscribe(c.gameChannel)
  if err != nil { return err }
  return nil
}

func (c *client) syncGame() error {
  var err error
  var game *api.GameState
  c.notifier.Partial("Retrieving game state")
  game, err = c.remote.ShowGame(c.game.Key)
  if err != nil { return err }
  c.game = game
  if !c.registered {
    err = c.register()
    if err != nil { return err }
  }
  c.notifier.Partial("Saving game state")
  err = c.saveGame()
  if err != nil { return err }
  c.notifier.Partial("Retrieving blocks")
  err = c.store.GetChain(c.game.FirstBlock, c.game.LastBlock)
  if err != nil { return err }
  c.notifier.Final("The game is up to date")
  return nil
}

func (c *client) saveGame() (err error) {
  buf := new(bytes.Buffer)
  json.NewEncoder(buf).Encode(c.game)
  err = ioutil.WriteFile("game.json", buf.Bytes(), 0644)
  return
}

func (c *client) leaveGame() {
  // TODO: unsubscribe
  // TODO: remove game.json
  // TODO: clear block store
  c.game = nil
  c.gameChannel = ""
  c.registered = false
  c.playerRanks = nil
}

func (c *client) register() error {
  var err error
  c.notifier.Partial("Registering players")
  var ranks []uint32
  ranks, err = c.remote.Register(c.game.Key, uint32(len(c.players)))
  if err != nil { return err }
  c.registered = true
  c.playerRanks = ranks
  return nil
}
