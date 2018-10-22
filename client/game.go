
package client

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "os"
  "tezos-contests.izibi.com/tc-node/api"
)

func (cl *client) loadGame() error {
  var err error
  var b []byte
  filepath := "game.json"
  _, err = os.Stat(filepath)
  if os.IsNotExist(err) {
    cl.game = nil
    cl.gameChannel = ""
    return nil
  }
  b, err = ioutil.ReadFile(filepath)
  if err != nil { return err }
  game := new(api.GameState)
  err = json.NewDecoder(bytes.NewBuffer(b)).Decode(game)
  if err != nil { return err }
  cl.game = game
  cl.gameChannel = "game:" + game.Key
  err = cl.subscribe(cl.gameChannel)
  if err != nil { return err }
  return nil
}

func (cl *client) syncGame() (uint64, error) {
  var err error
  var game *api.GameState
  cl.notifier.Partial("Retrieving game state")
  game, err = cl.remote.ShowGame(cl.game.Key)
  if err != nil { return 0, err }
  cl.game = game
  if !cl.registered {
    err = cl.register()
    if err != nil { return 0, err }
  }
  cl.notifier.Partial("Saving game state")
  err = cl.saveGame()
  if err != nil { return 0, err }
  cl.notifier.Partial("Retrieving blocks")
  err = cl.store.GetChain(cl.game.FirstBlock, cl.game.LastBlock)
  if err != nil { return 0, err }
  var currentRound uint64
  currentRound, err = cl.lastRoundNumber()
  if err != nil { return 0, err }
  cl.notifier.Final(fmt.Sprintf("Up-to-date at round %d", currentRound))
  return currentRound, nil
}

func (cl *client) saveGame() (err error) {
  buf := new(bytes.Buffer)
  json.NewEncoder(buf).Encode(cl.game)
  err = ioutil.WriteFile("game.json", buf.Bytes(), 0644)
  return
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

func (cl *client) lastRoundNumber() (uint64, error) {
  if cl.game == nil { return 0, fmt.Errorf("no current game") }
  n, ok := cl.store.Index.GetRoundByHash(cl.game.LastBlock)
  if !ok { return 0, fmt.Errorf("no game state on current block") }
  return n, nil
}
