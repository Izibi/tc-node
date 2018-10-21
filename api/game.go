
package api

import (
  "fmt"
)

func (s *Server) NewGame(firstBlock string) (*GameState, error) {
  type Request struct {
    Author string `json:"author"`
    FirstBlock string `json:"first_block"`
  }
  res := GameState{}
  err := s.SignedRequest("/Games", Request{
    Author: s.Author(),
    FirstBlock: firstBlock,
  }, &res)
  if err != nil { return nil, err }
  return &res, nil
}

func (s *Server) ShowGame(gameKey string) (*GameState, error) {
  type Response struct {
    Game GameState `json:"game"`
  }
  var res = &Response{}
  err := s.GetRequest("/Games/"+gameKey, res)
  if err != nil { return nil, err }
  return &res.Game, nil
}

/* Register a number of players for our and retrieve their ranks in the game. */
func (s *Server) Register(gameKey string, nbPlayers uint32) ([]uint32, error) {
  type Request struct {
    Author string `json:"author"`
    NbPlayers uint32 `json:"nbPlayers"`
  }
  type Response struct {
    Ranks []uint32 `json:"ranks"`
  }
  reqPath := fmt.Sprintf("/Games/%s/Register", gameKey)
  var res Response
  err := s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    NbPlayers: nbPlayers,
  }, &res)
  if err != nil { return nil, err }
  return res.Ranks, nil
}

func (s *Server) InputCommands(gameKey string, currentBlock string, teamPlayer uint32, commands string) error {
  type Request struct {
    Author string `json:"author"`
    CurrentBlock string `json:"current_block"`
    Player uint32 `json:"player"`
    Commands string `json:"commands"`
  }
  reqPath := fmt.Sprintf("/Games/%s/Commands", gameKey)
  return s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    Player: teamPlayer,
    CurrentBlock: currentBlock,
    Commands: commands,
  }, nil)
}

func (s *Server) CloseRound(gameKey string, currentBlock string) (string, error) {
  type Request struct {
    Author string `json:"author"`
    CurrentBlock string `json:"current_block"`
  }
  type Response struct {
    NewBlock string `json:"new_block"`
  }
  var err error
  var res Response
  reqPath := fmt.Sprintf("/Games/%s/CloseRound", gameKey)
  err = s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    CurrentBlock: currentBlock,
  }, &res)
  if err != nil { return "", err }
  return res.NewBlock, nil
}
