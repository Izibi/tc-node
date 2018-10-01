
package api

import (
  "tezos-contests.izibi.com/game/keypair"
)

type InputCommandsRequest struct {
  GameKey string `json:"game_key"`
  TeamKey string `json:"team_key"`
  TeamPlayer uint32 `json:"team_player"`
  CurrentBlock string `json:"current_block"`
  Commands string `json:"commands"`
}

type EndRoundRequest struct {
  GameKey string `json:"game_key"`
  TeamKey string `json:"team_key"`
  CurrentBlock string `json:"current_block"`
}
type EndRoundResponse struct {
  NewBlock string `json:"new_block"`
}

func (s *Server) NewGame(firstBlock string) (*GameState, error) {
  type Request struct {
    FirstBlock string `json:"first_block"`
  }
  req := Request{
    FirstBlock: firstBlock,
  }
  res := GameState{}
  err := s.PlainRequest("/Games", &req, &res)
  if err != nil { return nil, err }
  return &res, nil
}

func (s *Server) ShowGame(gameKey string) (res *GameState, err error) {
  res = &GameState{}
  err = s.GetRequest("/Games/"+gameKey, res)
  if err != nil { return nil, err }
  return res, nil
}

func (s *Server) InputCommands (gameKey string, currentBlock string, teamKeyPair *keypair.KeyPair, teamPlayer uint32, commands string) error {
  return s.SignedRequest(teamKeyPair, "/games/commands", InputCommandsRequest{
    GameKey: gameKey,
    TeamKey: teamKeyPair.Public,
    TeamPlayer: teamPlayer,
    CurrentBlock: currentBlock,
    Commands: commands,
  }, nil)
}

func (s *Server) EndRound (gameKey string, currentBlock string, teamKeyPair *keypair.KeyPair) (string, error) {
  var err error
  var res EndRoundResponse
  err = s.SignedRequest(teamKeyPair, "/games/end_round", EndRoundRequest{
    GameKey: gameKey,
    TeamKey: teamKeyPair.Public,
    CurrentBlock: currentBlock,
  }, &res)
  if err != nil { return "", err }
  return res.NewBlock, nil
}
