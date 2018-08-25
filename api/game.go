
package api

import (
  "tezos-contests.izibi.com/tezos-play/keypair"
)

type InputCommandsRequest struct {
  GameKey string `json:"game_key"`
  ValidForRound uint32 `json:"valid_for_round"`
  TeamKey string `json:"team_key"`
  TeamPlayer uint32 `json:"team_player"`
  Commands string `json:"commands"`
}

type EndRoundRequest struct {
  TeamKey string `json:"team_key"`
  GameKey string `json:"game_key"`
  Round uint32 `json:"round"`
}
type EndRoundResponse struct {
  NewBlock string `json:"new_block"`
}

func (s *Server) NewGame(gameParams GameParams) (res *GameState, err error) {
  res = &GameState{}
  err = s.PlainRequest("/games", gameParams, res)
  return
}

func (s *Server) ShowGame(gameKey string) (res *GameState, err error) {
  res = &GameState{}
  err = s.GetRequest("/games/"+gameKey, res)
  if err != nil { return nil, err }
  return res, nil
}

func (s *Server) InputCommands (gameKey string, round uint32, teamKeyPair *keypair.KeyPair, teamPlayer uint32, commands string) error {
  return s.SignedRequest(teamKeyPair, "/games/commands", InputCommandsRequest{
    GameKey: gameKey,
    ValidForRound: round,
    TeamKey: teamKeyPair.Public,
    TeamPlayer: teamPlayer,
    Commands: commands,
  }, nil)
}

func (s *Server) EndRound (gameKey string, round uint32, teamKeyPair *keypair.KeyPair) (string, error) {
  var err error
  var res EndRoundResponse
  err = s.SignedRequest(teamKeyPair, "/games/end_round", EndRoundRequest{
    TeamKey: teamKeyPair.Public,
    GameKey: gameKey,
    Round: round,
  }, &res)
  if err != nil { return "", err }
  return res.NewBlock, nil
}
