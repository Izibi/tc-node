
package main

import (
  "path/filepath"
  "flag"
  "fmt"
  "ioutil"
  "os"

  "gopkg.in/yaml.v2"
  "tezos-contests.izibi.com/tezos-play/api"
  "tezos-contests.izibi.com/tezos-play/keypair"
  //"tezos-contests.izibi.com/tezos-play/block_store"
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
}

func run() error {
  var err error

  var configFile []byte
  configFile, err = ioutil.ReadFile("config.yaml") ([]byte, error)
  if err != nil { return err }
  var config Config
  err = yaml.Unmarshal(configFile, config)
  if err != nil { return err }

  if config.StoreCacheDir == "" {
    config.StoreCacheDir = "store"
  }
  config.StoreCacheDir, err := filepath.Abs("store")
  if err != nil { return err }

  remote := api.New(config.ApiBaseUrl)
  // store := block_store.New(config.StoreBaseUrl, config.StoreCacheDir)

  var teamKeyPair *keypair.KeyPair
  teamKeyPair, err = keypair.Read("team.json")
  if err != nil { return err }

  chainHash, err := remote.NewChain(api.TaskParams{
      NbPlayers: 1,
      MapSide: 11,
    },
    "val start_skeleton : int * int -> unit\nval grow_skeleton : int * int -> unit\nval echo : string -> unit",
    "let start_skeleton (x, y) =\n  try\n    if Task.try_start_skeleton (x, y) then\n      print_string \"start_skeleton succeeded\\n\"\n    else\n      print_string \"start_skeleton returned false\\n\"\n  with ex ->\n    print_string \"start_skeleton raised an exception\";\n    print_string (Printexc.to_string ex);\n    print_string \"\\n\"\n\nlet grow_skeleton (x, y) =\n  try\n    if Task.try_grow_skeleton (x, y) then\n      print_string \"grow_skeleton succeeded\\n\"\n    else\n      print_string \"grow_skeleton returned false\\n\"\n  with ex ->\n    print_string \"grow_skeleton raised an exception\";\n    print_string (Printexc.to_string ex);\n    print_string \"\\n\"\nlet echo s = print_string s; print_string \"\n\"",
  )
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "chain  %s\n", chainHash)

  /*
  _, err = store.Get(chainHash)
  if err != nil { return err }
  */

  gameKey, err := remote.NewGame(chainHash, 10, 60, 2);
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "%s/%s\n", config.WatchGameUrl, gameKey)

  err = remote.InputCommands(teamKeyPair, gameKey, 1, 1,
    "start_skeleton (0, 5); echo \"ready\ngo!\"; grow_skeleton (1, 5); grow_skeleton (2, 5)")
  if err != nil { return err }

  err = remote.InputCommands(teamKeyPair, gameKey, 1, 2,
    "start_skeleton (5, 0); grow_skeleton (5, 1); echo \"thinking...\"; grow_skeleton (5, 2)")
  if err != nil { return err }

  h1, err := remote.EndRound(teamKeyPair, gameKey, 1)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "round1 %s\n", h1)

  h2, err := remote.EndRound(teamKeyPair, gameKey, 2)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "round2 %s\n", h2)

  return nil;
}
