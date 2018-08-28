
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
  KeypairFilename string `yaml:"keypair"`
  WatchGameUrl string `yaml:"watch_game_url"`
  NewGameParams api.GameParams `yaml:"new_game_params"`
  Players []PlayerConfig `yaml:"my_players"`
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

func LoadProtocol() (intf string, impl string, err error) {
  var b []byte
  b, err = ioutil.ReadFile("protocol.mli")
  if err != nil { return "", "", err }
  intf = string(b)
  b, err = ioutil.ReadFile("protocol.ml")
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
  if config.KeypairFilename == "" {
    config.KeypairFilename = "team.json"
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
  case "keypair":
    err = generateKeypair()
  case "new":
    err = startGame() /* XXX rename */
  case "join": /* TODO: and run */
    err = joinGame(flag.Arg(1))
  case "next":
    err = endOfRound()
  case "run":
    /* signal server to start timed rounds */
  default:
    err = errors.New("unknown command")
  }
  if err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
  }
}

func generateKeypair() (err error) {
  var kp *keypair.KeyPair
  kp, err = keypair.New()
  if err != nil { return err}
  err = kp.Write(config.KeypairFilename)
  return
}

func startGame() error {
  var err error
  var intf string
  var impl string
  fmt.Fprintf(os.Stderr, "Loading protocol\n")
  intf, impl, err = LoadProtocol()
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Sending protocol\n")
  var protoHash string
  protoHash, err = remote.NewProtocol(intf, impl)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Creating game\n")
  game, err = remote.NewGame(config.NewGameParams, protoHash);
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Saving game state\n")
  err = SaveGame(game)
  fmt.Fprintf(os.Stderr, "Clearing store\n")
  err = store.Clear()
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Retrieving blockchain\n")
  err = store.GetChain(game.CurrentBlock)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "open %s/%s\n", config.WatchGameUrl, game.Key)
  return nil
}

func joinGame(gameKey string) error {
  var err error
  fmt.Fprintf(os.Stderr, "Retrieving game state\n")
  game, err = remote.ShowGame(gameKey)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Saving game state\n")
  err = SaveGame(game)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Clearing store\n")
  err = store.Clear()
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Retrieving blockchain\n")
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

func endOfRound() (err error) {
  /* Load game */
  game, err = LoadGame()
  if err != nil { return }
  /* Load key pair */
  var teamKeyPair *keypair.KeyPair
  teamKeyPair, err = keypair.Read(config.KeypairFilename)
  if err != nil { return }
  /* Send commands */
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
  /* End round.  TODO: wait for end of round. */
  _, err = remote.EndRound(game.Key, game.CurrentRound, teamKeyPair)
  if err != nil { return }
  /* Retrieve updated game. */
  game, err = remote.ShowGame(game.Key)
  if err != nil { return }
  /* Retrieve current block. */
  _, err = store.Get(game.CurrentBlock)
  if err != nil { return }
  /* Save game state. */
  err = SaveGame(game)
  if err != nil { return }
  return
}
