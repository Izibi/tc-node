
package api

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "time"
  "strconv"
  "github.com/go-errors/errors"
  "tezos-contests.izibi.com/backend/signing"
)

func (s *Server) NewGame(firstBlock string) (*GameState, error) {
  type Request struct {
    Author string `json:"author"`
    FirstBlock string `json:"first_block"`
    Timestamp string `json:"timestamp"`
  }
  res := GameState{}
  err := s.SignedRequest("/Games", Request{
    Author: s.Author(),
    FirstBlock: firstBlock,
    Timestamp: time.Now().Format(time.RFC3339),
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
    GameKey string `json:"gameKey"`
    Action string `json:"action"`
    BotIds []uint32 `json:"botIds"`
  }
  type Response struct {
    Ranks []uint32 `json:"ranks"`
  }
  reqPath := fmt.Sprintf("/Games/%s", gameKey)
  var res Response
  err := s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    Action: "register bots",
    GameKey: gameKey,
    BotIds: botIds,
  }, &res)
  if err != nil { return nil, err }
  return res.Ranks, nil
}

func (s *Server) InputCommands(gameKey string, currentBlock string, teamPlayer uint32, commands string) error {
  type Request struct {
    Author string `json:"author"`
    GameKey string `json:"gameKey"`
    Action string `json:"action"`
    Player uint32 `json:"player"`
    CurrentBlock string `json:"current_block"`
    Commands string `json:"commands"`
  }
  reqPath := fmt.Sprintf("/Games/%s", gameKey)
  return s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    Action: "enter commands",
    GameKey: gameKey,
    Player: teamPlayer,
    CurrentBlock: currentBlock,
    Commands: commands,
  }, nil)
}

func (s *Server) CloseRound(gameKey string, currentBlock string) ([]byte, error) {
  type Request struct {
    Author string `json:"author"`
    GameKey string `json:"gameKey"`
    Action string `json:"action"`
    CurrentBlock string `json:"current_block"`
  }
  type Response struct {
    Commands json.RawMessage `json:"commands"`
  }
  var err error
  var res Response
  reqPath := fmt.Sprintf("/Games/%s", gameKey)
  err = s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    Action: "close round",
    GameKey: gameKey,
    CurrentBlock: currentBlock,
  }, &res)
  if err != nil { return nil, err }
  return res.Commands, nil
}

func (s *Server) Ping(gameKey string) (io.ReadCloser, error) {
  var err error
  type Request struct {
    Author string `json:"author"`
    GameKey string `json:"gameKey"`
    Action string `json:"action"`
    Timestamp string `json:"timestamp"`
  }
  b := new(bytes.Buffer)
  err = json.NewEncoder(b).Encode(&Request{
    Author: s.Author(),
    Action: "ping",
    GameKey: gameKey,
    Timestamp: unixMillisTimestamp(),
  })
  if err != nil { return nil, errors.Errorf("malformed message: %s", err) }
  bsReq, err := signing.Sign(s.teamKeyPair.Private, s.ApiKey, b.Bytes())
  var resp *http.Response
  url := fmt.Sprintf("%s/Games/%s", s.Base, gameKey)
  resp, err = http.Post(url, "text/plain", bytes.NewReader(bsReq))
  if err != nil { return nil, err }
  if resp.StatusCode < 200 || resp.StatusCode >= 299 {
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    return nil, errors.Errorf("%s: %s", resp.Status, buf.String())
  }
  return resp.Body, nil
}

func (s *Server) Pong(gameKey string, payload string, botIds []uint32) error {
  type Request struct {
    Author string `json:"author"`
    GameKey string `json:"gameKey"`
    Action string `json:"action"`
    Payload string `json:"payload"`
    BotIds []uint32 `json:"botIds"`
    Timestamp string `json:"timestamp"`
  }
  var err error
  reqPath := fmt.Sprintf("/Games/%s", gameKey)
  err = s.SignedRequest(reqPath, Request{
    Author: s.Author(),
    GameKey: gameKey,
    Action: "pong",
    BotIds: botIds,
    Payload: payload,
    Timestamp: unixMillisTimestamp(),
  }, nil)
  if err != nil { return err }
  return nil
}

func unixMillisTimestamp() string {
  return strconv.FormatInt(time.Now().UnixNano() / 1000000, 10)
}
