
package api

import (
  "tezos-contests.izibi.com/tezos-play/keypair"
)

type NewGameRequest struct {
  FirstBlock string `json:"first_block"`
  NbPlayers uint32 `json:"nb_players"`
  NbRounds uint32 `json:"nb_rounds"`
  RoundDuration uint32 `json:"round_duration"`
  CyclesPerRound uint32 `json:"cycles_per_round"`
  // Signature string `json:"signature"`
}
type PlayerInfos struct {
  Rank uint32 `json:"rank"`
  TeamKey string `json:"team_key"`
  TeamPlayer uint32 `json:"team_player"`
}
type ShowGameResponse struct {
  Key string `json:"key"`
  NbPlayers uint32 `json:"nb_players"`
  Player []PlayerInfos `json:"players"`
  NbRounds uint32 `json:"nb_rounds"`
  CurrentRound uint32 `json:"current_round"`
  RoundDuration uint32 `json:"round_duration"`
  EndOfRound string `json:"end_of_round"`
  CyclesPerRound uint32 `json:"cycles_per_round"`
  FirstBlock string `json:"first_block"`
  CurrentBlock string `json:"current_block"`
}

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

func (s *Server) NewGame(chainHash string, nbRounds uint32, roundDuration uint32, cyclesPerRound uint32) (res *ShowGameResponse, err error) {
  res = &ShowGameResponse{}
  err = s.PlainRequest("/games", NewGameRequest{
    FirstBlock: chainHash,
    NbRounds: nbRounds,
    RoundDuration: roundDuration,
    CyclesPerRound: cyclesPerRound,
  }, res)
  return
}

func (s *Server) ShowGame(gameKey string) (res *ShowGameResponse, err error) {
  res = &ShowGameResponse{}
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
