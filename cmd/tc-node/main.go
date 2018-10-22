
package main

import (
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "path/filepath"
  "time"

  "gopkg.in/yaml.v2"
  "github.com/eiannone/keyboard"
  "tezos-contests.izibi.com/backend/signing"
  "tezos-contests.izibi.com/tc-node/api"
  "tezos-contests.izibi.com/tc-node/block_store"
  "tezos-contests.izibi.com/tc-node/client"
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
  LastRoundCommandsSent uint64
  Latency time.Duration
  TimeDelta time.Duration
}

var config Config
var cl client.Client
var remote *api.Server
var store *block_store.Store
var notifier = &Notifier{}

func main() {

  var err error
  flag.Parse()
  cmd := flag.Args()

  /* Load the configuration file. */
  notifier.Partial("Loading config.yaml")
  err = Configure()
  if err != nil { panic(err) }

  /* Load the team's key pair */
  notifier.Partial("Loading the team's keypair")
  var teamKeyPair *signing.KeyPair
  teamKeyPair, err = loadKeyPair(config.KeypairFilename)
  if err != nil {
    teamKeyPair, err = generateKeypair()
    if err != nil {
      DangerFmt.Printf("failed to generate keypair: %v\n", err)
      os.Exit(0)
    }
    notifier.Final("")
    fmt.Print("A new keypair has been saved in ")
    SuccessFmt.Println(config.KeypairFilename)
    fmt.Printf("Your team's public key: \n    ")
    ImportantFmt.Printf("%s\n\n", teamKeyPair.Public)
    /* Exit because the new key must be associated with a team in the
       contest's web interface before we can proceed (will not be able
       to connect the event stream until the team key is recognized). */
    os.Exit(0)
  }
  notifier.Final(fmt.Sprintf("Team key: %s", teamKeyPair.Public))

  /* Connect to the API, set up the store, and initialize the game client. */
  remote = api.New(config.ApiBaseUrl, config.ApiKey, teamKeyPair)
  store = block_store.New(config.StoreBaseUrl, config.StoreCacheDir)
  cl = client.New(notifier, config.Task, remote, store, teamKeyPair, config.Players)

  /* Check the local time. */
  notifier.Partial("Checking the local time")
  err = checkTime()
  if err != nil {
    notifier.Error(err)
    os.Exit(0)
  }

  notifier.Partial("Connecting to the event stream")
  var ech <-chan interface{}
  ech, err = cl.Connect()
  if err != nil {
    notifier.Error(err)
    DangerFmt.Printf("\nFailed to connect to the event stream.\n\n")
    fmt.Printf("Did you link your public key (above) to your team?\n");
    os.Exit(0)
  }

  if len(cmd) == 0 {
    /* "tc-node" reloads the current game. */
    err = cl.LoadGame()
    if err != nil {
      notifier.Error(err)
      fmt.Printf("Use the new or join commands to recover.\n");
      os.Exit(0)
    }
    notifier.Final("Game loaded")
  } else {
    switch cmd[0] {
    case "new":
      /* "tc-node new" creates a new game */
      err = cl.NewGame(config.NewGameParams)
      if err != nil {
        notifier.Error(err)
        os.Exit(0)
      }
      notifier.Final("Game created")
      break
    case "join":
      /* "tc-node join GAME_KEY" joins the specified game */
      if len(cmd) < 2 {
        DangerFmt.Print("\nUsage: join GAME_KEY\n")
        os.Exit(0)
      }
      err = cl.JoinGame(cmd[1])
      if err != nil {
        notifier.Error(err)
        os.Exit(0)
      }
      notifier.Final("Game joined")
    default:
      DangerFmt.Print("\nwat.\n")
      os.Exit(0)
    }
  }
  fmt.Printf("Game key: ")
  GameKeyFmt.Println(cl.Game().Key)

  InteractiveLoop(ech)
  os.Exit(0)
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
  return nil
}

func InteractiveLoop(ech <-chan interface{}) {
  kch := keyboardChannel()
  wch := cl.Worker()
  defer keyboard.Close()

  wch<- client.AlwaysSendCommands()
  for {
    select {
      /* TODO: We get a block event when a new block has been downloaded;
         we should test whether the block is current, and keep waiting if not.
         We should also bail out on an end-of-game events.
       */
      case ev := <-ech:
        switch e := ev.(type) {
        case client.NewBlockEvent:
          wch<- client.SyncThenSendCommands()
        case client.LocalPlayerFeedback:
          // e.Step is "begin" | "run" | "send" | "ready"
          if e.Step == "ready" {
            fmt.Printf("Local player %d is ready\n", e.Player.Number)
          }
          if e.Err != nil {
            notifier.Error(e.Err)
          }
        case error:
          notifier.Error(e)
        default:
          fmt.Printf("event %v\n", ev)
        }
        // fmt.Printf("TODO: client worker <- send commands (if needed)\n")
      case kp := <-kch:
        switch kp.key {
        case 0:
          switch kp.ch {
            case 0:
              return
            case 'p', 'P':
              wch<- client.Ping().Signal()
            case 's':
              wch<- client.Sync().Signal()
            case 'S':
              wch<- client.SyncThenSendCommands().Signal()
            default:
              // fmt.Printf("ch '%c'\n", kp.ch)
          }
        case keyboard.KeyEsc, keyboard.KeyCtrlC:
          return
        case keyboard.KeySpace:
          wch<- client.EndOfRound().Signal()
        case keyboard.KeyEnter:
          fmt.Println("Enter")
          wch<- client.AlwaysSendCommands().Signal()
        default:
          fmt.Printf("key %v\n", kp.key)
        }
    }
  }
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

func checkTime() error {
  ts, err := cl.GetTimeStats()
  if err != nil { return err }
  // TODO: post results to an API for statistics?
  config.TimeDelta = ts.Delta
  config.Latency = ts.Latency
  return nil
}

/*
var feedback = func (player *PlayerConfig, source string, err error) {
  if err == nil {
    fmt.Printf("Local player %d is ready\n", player.Number)
  } else {
    fmt.Printf("Error for local player %d\n", player.Number)
    fmt.Printf("Player %d error\n", player.Number)
    switch source {
    case "run":
      fmt.Printf("Error running command \"%s\"\n", player.CommandLine)
      fmt.Println(err.Error())
    case "send":
      fmt.Printf("Input rejected by server\n")
      fmt.Println(err.Error())
    }
    if cl.remote.LastError != "" {
      fmt.Println(cl.remote.LastError)
    }
    if cl.remote.LastDetails != "" {
      fmt.Println(cl.remote.LastDetails)
    }
  }
}
*/
