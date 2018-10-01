
package api

import (
  "fmt"
  "errors"
)

func (s *Server) AddProtocolBlock(parentHash string, intf string, impl string) (string, error) {
  type Request struct {
    Interface string `json:"interface"`
    Implementation string `json:"implementation"`
  }
  type Response struct {
    Hash string `json:"hash"`
    Error string `json:"error"`
    Details string `json:"details"`
  }
  var err error
  req := Request{
    Interface: intf,
    Implementation: impl,
  }
  var res Response
  path := fmt.Sprintf("/Blocks/%s/Protocol", parentHash)
  err = s.PlainRequest(path, &req, &res)
  if err != nil { return "", err }
  if res.Error != "" {
    fmt.Printf("Error in protocol:\n%s\n%s\n", res.Error, res.Details)
    return "", errors.New("error in protocol")
  }
  return res.Hash, nil
}
