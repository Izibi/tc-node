
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
  TeamKey string `json:"team_key"`
  GameKey string `json:"game_key"`
  PlayerNumber uint32 `json:"player_number"`
  Commands string `json:"commands"`
  ValidForRound uint32 `json:"valid_for_round"`
}

type EndRoundRequest struct {
  TeamKey string `json:"team_key"`
  GameKey string `json:"game_key"`
  Round uint32 `json:"round"`
}
type EndRoundResponse struct {
  NewBlock string `json:"new_block"`
}

func (s *Server) NewGame(chainHash string, nbRounds uint32, roundDuration uint32, cyclesPerRound uint32) (string, error) {
  var err error
  var res ShowGameResponse
  err = s.PlainRequest("/games", NewGameRequest{
    FirstBlock: chainHash,
    NbRounds: nbRounds,
    RoundDuration: roundDuration,
    CyclesPerRound: cyclesPerRound,
  }, &res)
  if err != nil { return "", err }
  return res.Key, nil
}

func (s *Server) InputCommands (teamKeyPair *keypair.KeyPair, gameKey string, round uint32, player uint32, commands string) error {
  return s.SignedRequest(teamKeyPair, "/games/commands", InputCommandsRequest{
    TeamKey: teamKeyPair.Public,
    GameKey: gameKey,
    Commands: commands,
    ValidForRound: round,
    PlayerNumber: player,
  }, nil)
}

func (s *Server) EndRound (teamKeyPair *keypair.KeyPair, gameKey string, round uint32) (string, error) {
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
