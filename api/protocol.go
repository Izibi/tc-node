
package api

import (
  "fmt"
  "errors"
)

type NewProtocolRequest struct {
  Interface string `json:"interface"`
  Implementation string `json:"implementation"`
}
type NewProtocolResponse struct {
  Hash string `json:"hash"`
  Error string `json:"error"`
  Details string `json:"details"`
}

func (s *Server) NewProtocol(parentHash string, intf string, impl string) (string, error) {
  var err error
  var res NewProtocolResponse
  path := fmt.Sprintf("/Blocks/%s/Protocol", parentHash)
  err = s.PlainRequest(path, NewProtocolRequest{
    Interface: intf,
    Implementation: impl,
  }, &res)
  if err != nil { return "", err }
  if res.Error != "" {
    fmt.Printf("Error in protocol:\n%s\n%s\n", res.Error, res.Details)
    return "", errors.New("error in protocol")
  }
  return res.Hash, nil
}
