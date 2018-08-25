
package main

import (
  "bytes"
  "encoding/json"
  "errors"
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "path/filepath"

  "gopkg.in/yaml.v2"
  "tezos-contests.izibi.com/tezos-play/api"
  "tezos-contests.izibi.com/tezos-play/keypair"
  "tezos-contests.izibi.com/tezos-play/block_store"
)

func main() {
  flag.Parse()
  if err := run(); err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
  }
}

type Config struct {
  ApiBaseUrl string `yaml:"api_base"`
  StoreBaseUrl string `yaml:"store_base"`
  StoreCacheDir string `yaml:"store_dir"`
  WatchGameUrl string `yaml:"watch_game_url"`
  TaskParams api.TaskParams `yaml:"task_params"`
  GameParams api.GameParams `yaml:"game_params"`
}

func LoadGame() (res *api.GameState, err error) {
  var b []byte
  b, err = ioutil.ReadFile("game.json")
  if err != nil { return nil, err }
  err = json.NewDecoder(bytes.NewBuffer(b)).Decode(res)
  return
}

func SaveGame(game *api.GameState) (err error) {
  buf := new(bytes.Buffer)
  json.NewEncoder(buf).Encode(game)
  err = ioutil.WriteFile("game.json", buf.Bytes(), 0644)
  return
}

func LoadLibrary() (intf string, impl string, err error) {
  var b []byte
  b, err = ioutil.ReadFile("library.mli")
  if err != nil { return "", "", err }
  intf = string(b)
  b, err = ioutil.ReadFile("library.ml")
  if err != nil { return "", "", err }
  impl = string(b)
  return
}

func run() error {
  var err error

  var configFile []byte
  configFile, err = ioutil.ReadFile("config.yaml")
  if err != nil { return err }
  var config Config
  err = yaml.Unmarshal(configFile, &config)
  if err != nil { return err }
  fmt.Println("config %v", config)

  if config.StoreCacheDir == "" {
    config.StoreCacheDir = "store"
  }
  config.StoreCacheDir, err = filepath.Abs("store")
  if err != nil { return err }

  remote := api.New(config.ApiBaseUrl)
  store := block_store.New(config.StoreBaseUrl, config.StoreCacheDir)

  var teamKeyPair *keypair.KeyPair
  teamKeyPair, err = keypair.Read("team.json")
  if err != nil { return err }

  var intf string
  var impl string
  intf, impl, err = LoadLibrary()
  if err != nil {
    return errors.New("library source code is missing")
  }

  gameOk := false
  var game *api.GameState
  var chainHash string
  game, err = LoadGame()
  if err == nil && config.TaskParams == game.TaskParams {
    chainHash, err = remote.NewChain(config.TaskParams, intf, impl)
    if err != nil {
      return errors.New("error verifying chain")
    }
    if game.GameParams.FirstBlock != chainHash {
      return errors.New("current game is for a different chain")
    }
    gameOk = true
  } else {
    chainHash, err = remote.NewChain(config.TaskParams, intf, impl)
    if err != nil {
      return errors.New("error creating chain")
    }
    _, err = store.Get(chainHash)
    if err != nil { return err }
    config.GameParams.FirstBlock = chainHash
  }

  if !gameOk {
    game, err = remote.NewGame(config.GameParams);
    if err != nil { return err }
    err = SaveGame(game)
    if err != nil { return err }
  }

  if game == nil {
    return errors.New("game init failed")
  }
  fmt.Fprintf(os.Stderr, "%s/%s\n", config.WatchGameUrl, game.Key)

  err = remote.InputCommands(game.Key, 1, teamKeyPair, 1,
    "start_skeleton (0, 5); echo \"ready\ngo!\"; grow_skeleton (1, 5); grow_skeleton (2, 5)")
  if err != nil { return err }

  err = remote.InputCommands(game.Key, 1, teamKeyPair, 2,
    "start_skeleton (5, 0); grow_skeleton (5, 1); echo \"thinking...\"; grow_skeleton (5, 2)")
  if err != nil { return err }

  var h1 string
  h1, err = remote.EndRound(game.Key, 1, teamKeyPair)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "round1 %s\n", h1)

  game, err = remote.ShowGame(game.Key)
  if err != nil { return err }
  _, err = store.Get(game.CurrentBlock)
  if err != nil { return err }
  err = SaveGame(game)
  if err != nil { return err }

  h2, err := remote.EndRound(game.Key, 2, teamKeyPair)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "round2 %s\n", h2)

  game, err = remote.ShowGame(game.Key)
  if err != nil { return err }
  _, err = store.Get(game.CurrentBlock)
  err = SaveGame(game)
  if err != nil { return err }

  return nil;
}
