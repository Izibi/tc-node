package main

import (
  "fmt"
  "os"
  "tezos-contests.izibi.com/tezos-play/api"
  "tezos-contests.izibi.com/tezos-play/keypair"
)

func main() {
  if err := run(); err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
  }
}

func run() error {
  var err error

  var teamKeyPair *keypair.KeyPair
  teamKeyPair, err = keypair.Read("team.json")
  if err != nil { return err }

  remote := api.New("http://127.0.0.1:8100/task1/api")

  chainHash, err := remote.NewChain(api.TaskParams{
      NbTeams: 1,
      MapSide: 11,
    },
    "val start_skeleton : int * int -> unit\nval grow_skeleton : int * int -> unit",
    "let start_skeleton (x, y) =\n  try\n    if Task.try_start_skeleton (x, y) then\n      print_string \"start_skeleton succeeded\\n\"\n    else\n      print_string \"start_skeleton returned false\\n\"\n  with ex ->\n    print_string \"start_skeleton raised an exception\";\n    print_string (Printexc.to_string ex);\n    print_string \"\\n\"\n\nlet grow_skeleton (x, y) =\n  try\n    if Task.try_grow_skeleton (x, y) then\n      print_string \"grow_skeleton succeeded\\n\"\n    else\n      print_string \"grow_skeleton returned false\\n\"\n  with ex ->\n    print_string \"grow_skeleton raised an exception\";\n    print_string (Printexc.to_string ex);\n    print_string \"\\n\"\n",
  )
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "chain  %s\n", chainHash)

  gameKey, err := remote.NewGame(chainHash, 10, 60, 2);
  if err != nil { return err }
  fmt.Fprintf(os.Stderr, "game   %s\n", gameKey)

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
