
package main

import (
  "bytes"
  "encoding/json"
  "errors"
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "os/exec"
  "path/filepath"
  "strings"

  "gopkg.in/yaml.v2"
  "tezos-contests.izibi.com/tezos-play/api"
  "tezos-contests.izibi.com/tezos-play/keypair"
  "tezos-contests.izibi.com/tezos-play/block_store"
)

type Config struct {
  BaseUrl string `yaml:"base_url"`
  ApiBaseUrl string `yaml:"api_base"`
  StoreBaseUrl string `yaml:"store_base"`
  StoreCacheDir string `yaml:"store_dir"`
  WatchGameUrl string `yaml:"watch_game_url"`
  TaskParams api.TaskParams `yaml:"task_params"`
  GameParams api.GameParams `yaml:"game_params"`
  Players []PlayerConfig `yaml:"players"`
}

type PlayerConfig struct {
  Number uint32 `yaml:"number"`
  CommandLine string `yaml:"command_line"`
}

type CommandEnv struct {
  RoundNumber uint32
  PlayerNumber uint32
}

var config Config
var remote *api.Server
var game *api.GameState
var store *block_store.Store

func LoadGame() (*api.GameState, error) {
  var err error
  var b []byte
  b, err = ioutil.ReadFile("game.json")
  if err != nil { return nil, err }
  res := new(api.GameState)
  err = json.NewDecoder(bytes.NewBuffer(b)).Decode(res)
  if err != nil { return nil, err }
  return res, nil
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

func Configure() error {
  var err error
  var configFile []byte
  configFile, err = ioutil.ReadFile("config.yaml")
  if err != nil { return err }
  err = yaml.Unmarshal(configFile, &config)
  if err != nil { return err }
  if config.ApiBaseUrl == "" {
    config.ApiBaseUrl = config.BaseUrl + "/api"
  }
  if config.StoreBaseUrl == "" {
    config.StoreBaseUrl = config.BaseUrl + "/store"
  }
  if config.WatchGameUrl == "" {
    config.WatchGameUrl = config.BaseUrl + "/games"
  }
  if config.StoreCacheDir == "" {
    config.StoreCacheDir = "store"
  }
  config.StoreCacheDir, err = filepath.Abs(config.StoreCacheDir)
  if err != nil { return err }
  remote = api.New(config.ApiBaseUrl)
  store = block_store.New(config.StoreBaseUrl, config.StoreCacheDir)
  return nil
}

func main() {
  var err error
  flag.Parse()
  err = Configure()
  if err != nil {
    fmt.Fprintf(os.Stderr, "init error: %v\n", err)
    os.Exit(1)
  }
  switch cmd := flag.Arg(0); cmd {
  case "start":
    err = startGame()
  case "join":
    err = joinGame(flag.Arg(1))
  case "send":
    err = sendCommands()
  case "next":
    err = endOfRound()
  default:
    err = errors.New("unknown command")
  }
  if err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
  }
}

func startGame() error {
  var err error
  var intf string
  var impl string
  intf, impl, err = LoadLibrary()
  if err != nil {
    return errors.New("library source code is missing")
  }
  var chainHash string
  game, err = LoadGame()
  chainHash, err = remote.NewChain(config.TaskParams, intf, impl)
  if err != nil {
    return errors.New("error creating chain")
  }
  _, err = store.Get(chainHash)
  if err != nil { return err }
  config.GameParams.FirstBlock = chainHash
  game, err = remote.NewGame(config.GameParams);
  if err != nil { return err }
  err = SaveGame(game)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "%s/%s\n", config.WatchGameUrl, game.Key)
  store.Clear()
  return nil
}

func joinGame(gameKey string) error {
  var err error
  game, err = remote.ShowGame(gameKey)
  if err != nil { return err }
  err = SaveGame(game)
  if err != nil { return err }
  store.Clear()
  err = store.GetChain(game.CurrentBlock)
  if err != nil { return err }
  return nil
}

func runCommand(shellCmd string, env CommandEnv) (string, error) {
  cmd := exec.Command("sh", "-c", shellCmd)
  cmd.Env = append(os.Environ(),
    fmt.Sprintf("ROUND_NUMBER=%d", env.RoundNumber),
    fmt.Sprintf("PLAYER_NUMBER=%d", env.PlayerNumber),
  )
  input := fmt.Sprintf("%d %d", env.RoundNumber, env.PlayerNumber)
  cmd.Stdin = strings.NewReader(input)
  var out bytes.Buffer
  cmd.Stdout = &out
  err := cmd.Run()
  if err != nil { return "", err }
  return out.String(), nil
}

func sendCommands() (err error) {

  game, err = LoadGame()
  if err != nil { return }

  var teamKeyPair *keypair.KeyPair
  teamKeyPair, err = keypair.Read("team.json")
  if err != nil { return }

  for _, player := range config.Players {
    var commands string
    commands, err = runCommand(player.CommandLine, CommandEnv{
      RoundNumber: game.CurrentRound,
      PlayerNumber: player.Number,
    })
    if err != nil { return }
    err = remote.InputCommands(
      game.Key, game.CurrentRound, teamKeyPair, uint32(player.Number),
      commands)
    if err != nil { return }
  }

  return
}

func endOfRound() (err error) {
  game, err = LoadGame()
  if err != nil { return }
  var teamKeyPair *keypair.KeyPair
  teamKeyPair, err = keypair.Read("team.json")
  if err != nil { return }
  _, err = remote.EndRound(game.Key, game.CurrentRound, teamKeyPair)
  if err != nil { return }
  game, err = remote.ShowGame(game.Key)
  if err != nil { return }
  _, err = store.Get(game.CurrentBlock)
  if err != nil { return }
  err = SaveGame(game)
  if err != nil { return }
  return
}
