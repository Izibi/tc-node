
package api

import (
  "fmt"
  "github.com/go-errors/errors"
)

func (srv *Server) Subscribe(key string, channel string) error {
  fmt.Printf("subscribing to %s\n", channel)
  var err error
  type Request struct {
    Subscribe []string `json:"subscribe"`
  }
  type Result struct {
    Result bool `json:"result"`
    Error string `json:"error"`
  }
  req := Request{Subscribe: []string{channel}}
  var res Result
  err = srv.PlainRequest("/Events/" + key, req, &res)
  if err != nil { return err }
  if res.Error != "" { return errors.New(res.Error) }
  return nil
}
