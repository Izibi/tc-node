
/*

  See https://app.swaggerhub.com/apis/epixode1/BYhZzCNUCkA/4.0.0

*/

package api

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
  CreatedAt string `json:"createdAt"`
  UpdatedAt string `json:"updatedAt"`
  OwnerId string `json:"ownerId"`
  FirstBlock string `json:"firstBlock"`
  LastBlock string `json:"lastBlock"`
  StartedAt *string `json:"startedAt"`
  RoundEndsAt *string `json:"roundEndsAt"`
  IsLocked bool `json:"isLocked"`
  NbCyclesPerRound uint `json:"nbCyclesPerRound"`
  CurrentRound uint32 `json:"currentRound"`
}

type AnyBlock struct {
  Type string `json:"type"`
  Parent string `json:"parent"`
  Round uint32 `json:"round"`
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
