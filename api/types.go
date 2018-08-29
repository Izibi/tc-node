
/*

  See https://app.swaggerhub.com/apis/epixode1/BYhZzCNUCkA/3.0.0

*/

package api

var Version string = "3.0.0"

/* This is specific to task1. */
type GameParams struct {
  NbPlayers uint32 `json:"nb_players" yaml:"nb_players"`
  MapSide uint32 `json:"map_side" yaml:"map_side"`
  FirstBlock string `json:"first_block"`
  NbRounds uint32 `json:"nb_rounds" yaml:"nb_rounds"`
  RoundDuration uint32 `json:"round_duration" yaml:"round_duration"`
  CyclesPerRound uint32 `json:"cycles_per_round" yaml:"cycles_per_round"`
}

type PlayerInfos struct {
  Rank uint32 `json:"rank"`
  TeamKey string `json:"team_key"`
  TeamPlayer uint32 `json:"team_player"`
}

type GameState struct {
  Key string `json:"key"`
  Player []PlayerInfos `json:"players"`
  GameParams GameParams `json:"game_params"`
  CurrentRound uint32 `json:"current_round"`
  CurrentBlock string `json:"current_block"`
  EndOfRound string `json:"end_of_round"` /* datetime */
}

type AnyBlock struct {
  Type string `json:"type"`
  Parent string `json:"parent"`
  Sequence uint32 `json:"sequence"`
}

type ProtocolBlock struct {
  AnyBlock
  Interface string `json:"interface"`
  Implementation string `json:"implementation"`
}

type SetupBlock struct {
  AnyBlock
  GameParams GameParams `json:"game_params"`
}

type PlayerCommand struct {
  PlayerRank uint32 `json:"player_rank"`
  Command string `json:"command"`
}

type CommandBlock struct {
  AnyBlock
  Commands [][]PlayerCommand `json:"commands"`
}
