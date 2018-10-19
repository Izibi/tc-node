
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
  // noticeFmt.Println("Retrieving game state")
  game, err = c.remote.ShowGame(c.game.Key)
  if err != nil { return err }
  c.game = game
  // noticeFmt.Println("Saving game state")
  err = c.saveGame()
  if err != nil { return err }
  // noticeFmt.Println("Retrieving blocks")
  err = c.store.GetChain(c.game.FirstBlock, c.game.LastBlock)
  if err != nil { return err }
  // successFmt.Println("Game is up to date.")
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
}
