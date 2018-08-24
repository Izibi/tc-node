package main

import (
  "path/filepath"
  "flag"
  "fmt"
  "os"
  "tezos-contests.izibi.com/tezos-play/api"
  "tezos-contests.izibi.com/tezos-play/keypair"
  "tezos-contests.izibi.com/tezos-play/block_store"
)

func main() {
  /*
  initCommand := flag.NewFlagSet("count", flag.ExitOnError)
  textPtr := flag.String("text", "", "Text to parse.")
  metricPtr := flag.String("metric", "chars", "Metric {chars|words|lines};.")
  uniquePtr := flag.Bool("unique", false, "Measure unique values of a metric.")
  */

  flag.Parse()
  if err := run(); err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
  }
}

type Config struct {
  ApiBaseUrl string
  StoreBaseUrl string
  StoreCacheDir string
  WatchGameUrl string
}

func run() error {
  var err error

  config := Config{
    ApiBaseUrl: "http://127.0.0.1:8100/task1/api",
    StoreBaseUrl: "http://127.0.0.1:8100/task1/store",
    WatchGameUrl: "http://localhost:8100/task1/games",
  }
  config.StoreCacheDir, err = filepath.Abs("store")
  if err != nil { return err }

  remote := api.New(config.ApiBaseUrl)
  store := block_store.New(config.StoreBaseUrl, config.StoreCacheDir)

  var teamKeyPair *keypair.KeyPair
  teamKeyPair, err = keypair.Read("team.json")
  if err != nil { return err }

  chainHash, err := remote.NewChain(api.TaskParams{
      NbTeams: 1,
      MapSide: 11,
    },
    "val start_skeleton : int * int -> unit\nval grow_skeleton : int * int -> unit",
    "let start_skeleton (x, y) =\n  try\n    if Task.try_start_skeleton (x, y) then\n      print_string \"start_skeleton succeeded\\n\"\n    else\n      print_string \"start_skeleton returned false\\n\"\n  with ex ->\n    print_string \"start_skeleton raised an exception\";\n    print_string (Printexc.to_string ex);\n    print_string \"\\n\"\n\nlet grow_skeleton (x, y) =\n  try\n    if Task.try_grow_skeleton (x, y) then\n      print_string \"grow_skeleton succeeded\\n\"\n    else\n      print_string \"grow_skeleton returned false\\n\"\n  with ex ->\n    print_string \"grow_skeleton raised an exception\";\n    print_string (Printexc.to_string ex);\n    print_string \"\\n\"\n",
  )
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "chain  %s\n", chainHash)
  _, err = store.Get(chainHash)
  if err != nil { return err }

  gameKey, err := remote.NewGame(chainHash, 10, 60, 2);
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "%s/%s\n", config.WatchGameUrl, gameKey)

  // const keys = ssbKeys.loadOrCreateSync("team-1");

  err = remote.InputTeamCommands(teamKeyPair, gameKey, 1,
    "start_skeleton (0, 5); grow_skeleton (1, 5); grow_skeleton (2, 5)")
  if err != nil { return err }

  h1, err := remote.EndRound(teamKeyPair, gameKey, 1)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "round1 %s\n", h1)

  h2, err := remote.EndRound(teamKeyPair, gameKey, 2)
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "round1 %s\n", h2)

  return nil;
}
