
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
  "github.com/k0kubun/go-ansi"
  "github.com/c-bata/go-prompt"
  "github.com/eiannone/keyboard"
  "tezos-contests.izibi.com/backend/signing"
  "tezos-contests.izibi.com/tc-node/api"
  "tezos-contests.izibi.com/tc-node/block_store"
  "tezos-contests.izibi.com/tc-node/client"
  "tezos-contests.izibi.com/tc-node/ui"
)

type Config struct {
  BaseUrl string `yaml:"base_url"`
  ApiBaseUrl string `yaml:"api_base"`
  StoreBaseUrl string `yaml:"store_base"`
  StoreCacheDir string `yaml:"store_dir"`
  ApiKey string `yaml:"api_key"`
  Task string `yaml:"task"`
  KeypairFilename string `yaml:"signing"`
  WatchGameUrl string `yaml:"watch_game_url"`
  NewGameParams map[string]interface{} `yaml:"new_game_params"`
  Players []client.PlayerConfig `yaml:"players"`
  EagerlySendCommands bool
  LastRoundCommandsSend uint64
  Latency time.Duration
  TimeDelta time.Duration
}

type Notifier struct{
  partial bool
  errorShown bool
}

var config Config
var cl client.Client
var remote *api.Server
var notifier = &Notifier{}

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
    notifier.errorShown = false
    err = nil
    switch cmd[0] {
    case "new":
      lastCommand = []string{"next"}
      err = cl.NewGame(config.NewGameParams)
      if err == nil {
        game := cl.Game()
        fmt.Printf("Game key: ")
        ui.SuccessFmt.Println(game.Key)
        _ = eagerlySendCommands()
      }
      // Client reported the error, no display
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
          ui.DangerFmt.Println("failed")
        } else {
          ui.SuccessFmt.Println("ok")
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
    if err != nil && !notifier.errorShown {
      notifier.Error(err)
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
    fmt.Println()
    fmt.Print("A new keypair has been saved in ")
    ui.SuccessFmt.Println(config.KeypairFilename)
    fmt.Printf("Your team's public key: \n    ")
    ui.ImportantFmt.Printf("%s\n\n", teamKeyPair.Public)
    /* Exit because the new key must be associated with a team in the
       contest's web interface before we can proceed (will not be able
       to connect the event stream until the team key is recognized). */
    os.Exit(0)
  }
  fmt.Printf("Team key: %s\n", teamKeyPair.Public)
  remote = api.New(config.ApiBaseUrl, config.ApiKey, teamKeyPair)
  store := block_store.New(config.StoreBaseUrl, config.StoreCacheDir)
  cl = client.New(config.Task, remote, store, teamKeyPair)
  cl.SetNotifier(notifier)
  /* Check the local time. */
  err = checkTime()
  if err != nil { return err }
  /* Start the client (will connect events, load game, and sync). */
  err = cl.Start()
  if err != nil { return err }
  game := cl.Game()
  if game != nil {
    fmt.Print("Game key: ")
    ui.GameKeyFmt.Println(game.Key)
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
  ts, err := cl.GetTimeStats()
  if err != nil { return err }
  // TODO: post results to an API for statistics?
  config.TimeDelta = ts.Delta
  config.Latency = ts.Latency
  return nil
}

func sendCommands() error {
  var err error
  var currentRound = cl.Game().CurrentRound
  fmt.Printf("Sending commands for round %d\n", currentRound + 1)
  if len(config.Players) == 0 {
    ui.WarningFmt.Println("no players configured!")
    return nil
  }
  var feedback = func (player *client.PlayerConfig, source string, err error) {
    if err == nil {
      ui.SuccessFmt.Printf("Player %d is ready\n", player.Number)
    } else {
      ui.DangerFmt.Printf("Player %d error\n", player.Number)
      switch source {
      case "run":
        fmt.Printf("Error running command \"%s\"\n", player.CommandLine)
        fmt.Println(err.Error())
      case "send":
        fmt.Printf("Input rejected by server\n")
        fmt.Println(err.Error())
      }
      if remote.LastError != "" {
        fmt.Println(remote.LastError)
      }
      if remote.LastDetails != "" {
        fmt.Println(remote.LastDetails)
      }
    }
  }
  var retry bool
  for {
    retry, err = cl.SendCommands(config.Players, feedback)
    if !retry {
      return err
    }
    err = cl.SyncGame()
    if err != nil {
      return err
    }
  }
  if err != nil {
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
        cl.Silence()
        return false
      }
  }
  cl.Silence()
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

func (n *Notifier) Partial(msg string) {
  ansi.EraseInLine(1)
  ansi.CursorHorizontalAbsolute(0)
  fmt.Print(msg)
  n.partial = true
}

func (n *Notifier) Final(msg string) {
  if n.partial {
    ansi.EraseInLine(1)
    ansi.CursorHorizontalAbsolute(0)
    fmt.Println(msg)
  }
  n.partial = false
}

func (n *Notifier) Error(err error) {
  if n.partial {
    ansi.EraseInLine(1)
    ansi.CursorHorizontalAbsolute(0)
    n.partial = false
  }
  ui.DangerFmt.Println(err.Error())
  n.errorShown = true
}
