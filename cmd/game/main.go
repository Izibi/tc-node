
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
  "github.com/fatih/color"
  "tezos-contests.izibi.com/game/api"
  "tezos-contests.izibi.com/game/block_store"
  "tezos-contests.izibi.com/backend/signing"
  "tezos-contests.izibi.com/game/sse"
)

var success = color.New(color.Bold, color.FgGreen)
var danger = color.New(color.Bold, color.FgRed)

type Config struct {
  BaseUrl string `yaml:"base_url"`
  ApiBaseUrl string `yaml:"api_base"`
  StoreBaseUrl string `yaml:"store_base"`
  StoreCacheDir string `yaml:"store_dir"`
  ApiKey string `yaml:"api_key"`
  Task string `yaml:"task"`
  KeypairFilename string `yaml:"signing"`
  WatchGameUrl string `yaml:"watch_game_url"`
  NewTaskParams map[string]interface{} `yaml:"new_task_params"`
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
    config.ApiBaseUrl = config.BaseUrl + "/backend"
  }
  if config.StoreBaseUrl == "" {
    config.StoreBaseUrl = config.BaseUrl + "/backend/Blocks"
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
  /* Load key pair */
  var teamKeyPair *signing.KeyPair
  teamKeyPair, _ = loadKeyPair(config.KeypairFilename)
  remote = api.New(config.ApiBaseUrl, config.ApiKey, teamKeyPair)
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
  case "time":
    err = getServerTime()
  case "new":
    err = startGame() /* XXX rename */
  case "join": /* TODO: and run */
    err = joinGame(flag.Arg(1))
  case "sync":
    err = syncGame()
  case "send":
    err = sendCommands()
  case "next":
    err = endOfRound()
  case "run":
    /* signal server to start timed rounds */
  default:
    err = errors.New("unknown command")
  }
  if err != nil {
    danger.Fprintln(os.Stderr, "Command failed.")
    fmt.Println(err)
    os.Exit(1)
  }
}

func generateKeypair() (err error) {
  var kp *signing.KeyPair
  kp, err = signing.NewKeyPair()
  if err != nil { return err }
  file, err := os.OpenFile(config.KeypairFilename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
  if err != nil { return err }
  defer file.Close()
  err = kp.Write(file)
  if err != nil { return err }
  return nil
}

func loadKeyPair (filename string) (*signing.KeyPair, error) {
  var err error
  f, err := os.Open(filename)
  if err != nil {
    return nil, err
  }
  defer f.Close()
  return signing.ReadKeyPair(f)
}

func getServerTime() (err error) {
  var time string
  time, err = remote.GetTime()
  if err != nil { return }
  fmt.Fprintf(os.Stderr, "Server time: %s\n", time)
  /* TODO print current time, delta, complain if >0.1s difference */
  return
}

func startGame() error {
  var err error
  var intf string
  var impl string
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Loading protocol\n")
  intf, impl, err = LoadProtocol()
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Sending protocol\n")
  protoHash, err := remote.AddProtocolBlock(config.Task, intf, impl)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Performing task setup\n")
  setupHash, err := remote.AddSetupBlock(protoHash, config.NewTaskParams)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Creating game\n")
  game, err = remote.NewGame(setupHash);
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Saving game state\n")
  err = SaveGame(game)
  fmt.Fprintf(os.Stderr, "Clearing store\n")
  err = store.Clear()
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Retrieving blocks\n")
  err = store.GetChain(game.FirstBlock, game.LastBlock)
  if err != nil { return err }
  fmt.Printf("Game key: ")
  success.Println(game.Key)
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
  fmt.Fprintf(os.Stderr, "Retrieving blocks\n")
  err = store.GetChain(game.FirstBlock, game.LastBlock)
  if err != nil { return err }
  success.Println("Success")
  return nil
}

func syncGame() error {
  game, err := LoadGame()
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Retrieving game state\n")
  game, err = remote.ShowGame(game.Key)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Saving game state\n")
  err = SaveGame(game)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "Retrieving blocks\n")
  err = store.GetChain(game.FirstBlock, game.LastBlock)
  if err != nil { return err }
  success.Println("Game is up to date.")
  ch, err := sse.EventSink(remote.Base + "/Events") // TODO: remote.Events()
  if err != nil { return err }
  e := <-ch
  if e.Type != "key" { return errors.New("expected key event") }
  key := e.Data
  err = remote.Subscribe(key, "games/" + game.Key)
  if err != nil { return err }
  e = <-ch
  fmt.Println("event! %v\n", e)
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
  /* Load game */
  if game == nil {
    game, err = LoadGame()
    if err != nil { return }
  }
  /* Send commands */
  for _, player := range config.Players {
    var commands string
    commands, err = runCommand(player.CommandLine, CommandEnv{
      RoundNumber: game.CurrentRound,
      PlayerNumber: player.Number,
    })
    if err != nil { return }
    err = remote.InputCommands(
      game.Key, game.LastBlock, uint32(player.Number),
      commands)
    if err != nil { return }
  }
  return
}

func endOfRound() (err error) {
  err = sendCommands()
  if err != nil { return }
  /* Close round. */
  _, err = remote.CloseRound(game.Key, game.LastBlock)
  if err != nil { return }
  /* TODO: wait for next block */
  /* Retrieve updated game. */
  game, err = remote.ShowGame(game.Key)
  if err != nil { return }
  /* Retrieve current block. */
  _, err = store.Get(game.LastBlock)
  if err != nil { return }
  /* Save game state. */
  err = SaveGame(game)
  if err != nil { return }
  return
}
