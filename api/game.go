
package api

import (
  "bytes"
  "encoding/json"
  "io"
  "fmt"
  "net/http"
  "github.com/go-errors/errors"
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

/* Register a number of bots for our team and retrieve their ranks in the game. */
func (s *Server) Register(gameKey string, botIds []uint32) ([]uint32, error) {
  type Request struct {
    Author string `json:"author"`
    Ids []uint32 `json:"ids"`
  }
  type Response struct {
    Ranks []uint32 `json:"ranks"`
  }
  reqPath := fmt.Sprintf("/Games/%s/Register", gameKey)
  var res Response
  err := s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    Ids: botIds,
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

func (s *Server) CloseRound(gameKey string, currentBlock string) ([]byte, error) {
  type Request struct {
    Author string `json:"author"`
    CurrentBlock string `json:"current_block"`
  }
  type Response struct {
    Commands json.RawMessage `json:"commands"`
  }
  var err error
  var res Response
  reqPath := fmt.Sprintf("/Games/%s/CloseRound", gameKey)
  err = s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    CurrentBlock: currentBlock,
  }, &res)
  if err != nil { return nil, err }
  return res.Commands, nil
}

func (s *Server) Ping(gameKey string) (io.ReadCloser, error) {
  var err error
  var resp *http.Response
  url := fmt.Sprintf("%s/Games/%s/Ping", s.Base, gameKey)
  resp, err = http.Post(url, "text/plain", bytes.NewReader([]byte{}))
  if err != nil { return nil, err }
  if resp.StatusCode < 200 || resp.StatusCode >= 299 {
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    return nil, errors.Errorf("%s: %s", resp.Status, buf.String())
  }
  return resp.Body, nil
}

func (s *Server) Pong(gameKey string, payload string) error {
  type Request struct {
    Author string `json:"author"`
    Payload string `json:"payload"`
  }
  var err error
  reqPath := fmt.Sprintf("/Games/%s/Pong", gameKey)
  err = s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    Payload: payload,
  }, nil)
  if err != nil { return err }
  return nil
}
