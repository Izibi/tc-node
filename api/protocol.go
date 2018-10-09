
package api

import (
  "fmt"
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
    return "", fmt.Errorf("Error in protocol:\n%s\n%s", res.Error, res.Details)
  }
  return res.Hash, nil
}
