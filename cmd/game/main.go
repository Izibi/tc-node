
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
  "github.com/eiannone/keyboard"
  "tezos-contests.izibi.com/game/api"
  "tezos-contests.izibi.com/game/block_store"
  "tezos-contests.izibi.com/backend/signing"
  "tezos-contests.izibi.com/game/client"
)

var successFmt = color.New(color.Bold, color.FgGreen)
var dangerFmt = color.New(color.Bold, color.FgRed)
var warningFmt = color.New(color.Bold, color.FgYellow)
var importantFmt = color.New(color.Bold, color.FgHiWhite)
var noticeFmt = color.New(color.FgHiBlack)
var gameKeyFmt = color.New(color.Bold, color.FgWhite)

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
  Players []client.PlayerConfig `yaml:"players"`
  EagerlySendCommands bool
  LastRoundCommandsSend uint64
}

var config Config
var cl client.Client

func main() {
  var err error
  /* Parse the command line and determine if running interactively, or a
     single command. */
  flag.Parse()
  cmd := flag.Args()
  interactive := len(cmd) == 0
  /* Load the configuration file. */
  err = Configure()
  if err != nil { panic(err) }
  /* Start up */
  err = Startup()
  if err != nil { panic(err) }
  /* display current status, start background sync?, etc */
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
    err = nil
    switch cmd[0] {
    case "new":
      lastCommand = []string{"next"}
      err = cl.NewGame(config.NewTaskParams)
      if err == nil {
        game := cl.Game()
        fmt.Printf("Game key: ")
        successFmt.Println(game.Key)
        _ = eagerlySendCommands()
      }
    case "join": /* TODO: and run */
      lastCommand = []string{"play"}
      var gameKey string
      if len(cmd) >= 2 {
        gameKey = cmd[1]
      } else {
        gameKey = prompt.Input("key> ", nil)
        gameKey = strings.TrimSpace(gameKey)
      }
      fmt.Printf("Joining game %s... ", gameKey)
      err = cl.JoinGame(gameKey)
      if err == nil {
        if err != nil {
          dangerFmt.Println("failed")
        } else {
          successFmt.Println("ok")
          _ = eagerlySendCommands()
        }
      }
    case "send":
      lastCommand = cmd
      _ = sendCommands()
    case "next":
      lastCommand = cmd
      // TODO: send commands if not sent
      err = cl.EndOfRound(config.Players)
      if err != nil { break }
      waitUntilNextRound()
      eagerlySendCommands()
    case "play":
      lastCommand = cmd
      err = playLoop()
    case "exit":
      os.Exit(0)
    }
    if err != nil {
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
    {Text: "new",     Description: "Create a new game"},
    {Text: "join",    Description: "Join and play an existing game"},
    {Text: "exit",    Description: "Exit the game CLI"},
  }
  if cl.Game() != nil {
    s = append(s, []prompt.Suggest{
      {Text: "play",    Description: "Play the current game"},
      {Text: "pause",   Description: "Pause the current game"},
      {Text: "send",    Description: "Resend commands for the next round"},
      {Text: "next",    Description: "Manually end the current round"},
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
  config.EagerlySendCommands = true
  return nil
}

func Startup() error {
  var err error
  /* Load key pair */
  var teamKeyPair *signing.KeyPair
  teamKeyPair, err = loadKeyPair(config.KeypairFilename)
  if err != nil {
    teamKeyPair, err = generateKeypair()
    if err != nil { return err }
    fmt.Print("A new keypair has been saved in ")
    successFmt.Println(config.KeypairFilename)
    fmt.Printf("Your team's public key: \n    ")
    importantFmt.Printf("%s\n\n", teamKeyPair.Public)
  } else {
    fmt.Printf("Team key: %s\n", teamKeyPair.Public)
  }
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
    _ = eagerlySendCommands()
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
  fmt.Print("Checking local time... ")
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

func sendCommands() error {
  var err error
  var currentRound = cl.Game().CurrentRound
  fmt.Printf("Sending commands for round %d... ", currentRound + 1)
  if len(config.Players) == 0 {
    warningFmt.Println("no 'players' configured!")
    return nil
  }
  var feedback = func (player *client.PlayerConfig, source string, err error) {
    if err == nil {
      successFmt.Printf("%d ", player.Number)
    } else {
      dangerFmt.Printf("%d\n", player.Number)
      fmt.Printf("Player %d: ", player.Number)
      if source == "run" {
        fmt.Printf("error running %s\n", player.CommandLine)
        dangerFmt.Println(err.Error())
      } else {
        fmt.Printf("commands rejected by server:\n")
        dangerFmt.Println(err.Error())
      }
    }
  }
  if cl.SendCommands(config.Players, feedback) {
    config.LastRoundCommandsSend = currentRound
    /* If the feedback function was never called with an error, the cursor is
       still at the end of the "Sending commands..." line, so add a newline. */
    fmt.Println("")
  }
  return err
}

/* Return true when a next round event has been received without error.
   Return false if interrupted by the user, or if an error occurred. */
func waitUntilNextRound() bool {
  keyboardChannel := make(chan struct{})
  go func() {
    var err error
    err = keyboard.Open()
    if err == nil {
      for {
        ch, key, err := keyboard.GetKey()
        if err != nil { break }
        if key == keyboard.KeyEsc { break }
        if ch == 0 && key == 0 && err == nil { break /* closed */ }
      }
    }
    keyboard.Close()
    close(keyboardChannel)
    return
  }()
  eventChannel := cl.Events()
  select {
    /* TODO: We get a block event when a new block has been downloaded;
       we should test whether the block is current, and keep waiting if not.
       We should also bail out on an end-of-game events.
     */
    case ev := <-eventChannel:
      newBlockEvent := ev.(*client.NewBlockEvent)
      if newBlockEvent != nil {
        fmt.Printf("Round %d has ended.\n", newBlockEvent.Round)
        /* Closing the keyboard lib will cause keyboard.GetKey to return. */
        keyboard.Close()
        break
      }
    case _, ok := <-keyboardChannel:
      if !ok {
        return false
      }
  }
  return true
}

func playLoop() error {
  var err error
  fmt.Println("Press Escape to stop playing the game.")
  var ok = true
  for ok {
    if cl.Game().CurrentRound != config.LastRoundCommandsSend {
      err = sendCommands()
      if err != nil { return err }
    }
    if !waitUntilNextRound() {
      break
    }
  }
  return nil
}

func eagerlySendCommands() error {
  if config.EagerlySendCommands {
    return sendCommands()
  }
  return nil
}
