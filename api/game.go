
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

func (s *Server) ShowGame(gameKey string) (res *GameState, err error) {
  res = &GameState{}
  err = s.GetRequest("/Games/"+gameKey, res)
  if err != nil { return nil, err }
  return res, nil
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
