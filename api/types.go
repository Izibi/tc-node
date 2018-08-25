
package api

type TaskParams struct {
  NbPlayers uint32 `json:"nb_players" yaml:"nb_players"`
  MapSide uint32 `json:"map_side" yaml:"map_side"`
}

type GameParams struct {
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
  TaskParams TaskParams `json:"task_params"`
  GameParams GameParams `json:"game_params"`
  CurrentRound uint32 `json:"current_round"`
  CurrentBlock string `json:"current_block"`
  EndOfRound string `json:"end_of_round"` /* datetime */
}

type FirstBlock struct {
  Protocol uint32 `json:"protocol"`
  Sequence uint32 `json:"sequence"`
  TaskParams TaskParams `json:"task_params"`
  LibraryInterface string `json:"library_interface"`
  LibraryImplementation string `json:"library_implementation"`
}
