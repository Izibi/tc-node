
package main

import (
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "path/filepath"
  "strings"
  "time"

  "gopkg.in/yaml.v2"
  "github.com/fatih/color"
  "github.com/k0kubun/go-ansi"
  "github.com/c-bata/go-prompt"
  "tezos-contests.izibi.com/game/api"
  "tezos-contests.izibi.com/game/block_store"
  "tezos-contests.izibi.com/backend/signing"
  "tezos-contests.izibi.com/game/client"
)

var successFmt = color.New(color.Bold, color.FgGreen)
var teamKeyFmt = color.New(color.Bold, color.FgMagenta)
var gameKeyFmt = color.New(color.Bold, color.FgYellow)
var dangerFmt = color.New(color.Bold, color.FgRed)
var noticeFmt = color.New(color.FgHiBlack)
var importantFmt = color.New(color.Bold, color.FgHiWhite)

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
  Players []client.PlayerConfig `yaml:"my_players"`
}

var config Config
var cl client.Client

func main() {
  var err error
  flag.Parse()
  err = Configure()
  if err != nil { panic(err) }
  /* display current status, start background sync?, etc */
  cmd := flag.Args()
  interactive := len(cmd) == 0
  var lastCommand = []string{""}
  if cl.Game() != nil {
    lastCommand = []string{"next"}
  }
  for interactive {
    if interactive {
      line := prompt.Input("> ", mainCommandCompleter)
      line = strings.TrimLeft(line, " ")
      cmd = strings.Split(line, " ")
    }
    if len(cmd) == 0 {
      break
    }
    if cmd[0] == "" {
      ansi.CursorPreviousLine(1)
      cmd = lastCommand
      fmt.Printf("> %s\n", strings.Join(cmd, " "))
    }
    switch cmd[0] {
    case "new":
      lastCommand = []string{"next"}
      err = cl.NewGame(config.NewTaskParams)
      game := cl.Game()
      if game != nil {
        fmt.Printf("Game key: ")
        successFmt.Println(game.Key)
      }
    case "join": /* TODO: and run */
      lastCommand = cmd
      var gameKey string
      if len(cmd) >= 2 {
        gameKey = cmd[1]
      } else {
        gameKey = prompt.Input("key> ", nil)
        gameKey = strings.TrimSpace(gameKey)
      }
      err = cl.JoinGame(gameKey)
    case "send":
      lastCommand = cmd
      err = cl.SendCommands(config.Players)
    case "next":
      lastCommand = cmd
      err = cl.EndOfRound(config.Players)
    case "play":
      lastCommand = cmd
      /* signal server to start timed rounds */
      // TODO: play!
    case "exit":
      os.Exit(0)
    }
    if err != nil {
      dangerFmt.Fprintln(os.Stderr, "Command failed.")
      fmt.Println(err)
      if !interactive {
        os.Exit(1)
      }
    }
  }
}

func mainCommandCompleter(d prompt.Document) []prompt.Suggest {
  input := d.TextBeforeCursor()
  input = strings.TrimLeft(input, " ")
  if strings.Contains(input, " ") {
    return []prompt.Suggest{}
  }
  s := []prompt.Suggest{
    {Text: "exit",    Description: "Exit the game CLI"},
    {Text: "join",    Description: "Join and play an existing game"},
    {Text: "new",     Description: "Create a new game"},
    {Text: "time",    Description: "Verify the local time"},
  }
  if cl.Game() != nil {
    s = append(s, []prompt.Suggest{
      {Text: "play",    Description: "Play the current game"},
      {Text: "next",    Description: "Manually end the current round"},
      {Text: "send",    Description: "Manually send commands for the next round"},
      {Text: "sync",    Description: "Update the current game"},
    }...)
  }
  return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
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
  teamKeyPair, err = loadKeyPair(config.KeypairFilename)
  if err != nil {
    teamKeyPair, err = generateKeypair()
    if err != nil { return err }
    fmt.Print("A new keypair has been generated in ")
    successFmt.Println(config.KeypairFilename)
  }
  fmt.Print("Team key: ")
  teamKeyFmt.Println(teamKeyPair.Public)
  remote := api.New(config.ApiBaseUrl, config.ApiKey, teamKeyPair)
  store := block_store.New(config.StoreBaseUrl, config.StoreCacheDir)
  cl = client.New(config.Task, remote, store, teamKeyPair)
  /* Check the local time. */
  err = checkTime()
  if err != nil { return err }
  /* Start the client (will connect events, load game, and sync). */
  cl.Start()
  game := cl.Game()
  if game != nil {
    fmt.Print("Game key: ")
    gameKeyFmt.Println(game.Key)
    fmt.Print("Running player code and sending commands... ")
    err = cl.SendCommands(config.Players)
    if err != nil {
      dangerFmt.Print("failed\n")
    } else {
      successFmt.Print("ok\n")
    }
  }
  return nil
}

func generateKeypair() (*signing.KeyPair, error) {
  var err error
  var kp *signing.KeyPair
  kp, err = signing.NewKeyPair()
  if err != nil { return nil, err }
  file, err := os.OpenFile(config.KeypairFilename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
  if err != nil { return nil, err }
  defer file.Close()
  err = kp.Write(file)
  if err != nil { return nil, err }
  return kp, nil
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

func checkTime() error {
  noticeFmt.Print("Checking local time... ")
  ts, err := cl.GetTimeStats()
  if err != nil { return err }
  delta := ts.Delta
  latency := ts.Latency
  sign := "+"
  if delta < 0 {
    delta = -delta
    sign = "-"
  }
  if delta >= 500 * time.Millisecond {
    dangerFmt.Println("error")
    noticeFmt.Printf("Local time: %s\n", ts.Local.Format(time.RFC3339))
    noticeFmt.Printf("Server time: %s\n", ts.Server.Format(time.RFC3339))
    noticeFmt.Printf("Latency: %s\n", latency.String())
    noticeFmt.Print("Difference: ")
    dangerFmt.Printf("%s%s\n", sign, delta.String())
    importantFmt.Println("\n    Please adjust your clock!\n")
  } else {
    successFmt.Println("ok")
  }
  // TODO: post results to an API for statistics?
  return nil
}
