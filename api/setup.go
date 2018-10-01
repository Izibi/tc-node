
package api

import (
  "fmt"
  "errors"
)

func (s *Server) AddSetupBlock(parentHash string, params map[string]interface{}) (string, error) {
  type Request struct {
    Params map[string]interface{} `json:"params"`
  }
  type Response struct {
    Hash string `json:"hash"`
    Error string `json:"error"`
    Details string `json:"details"`
  }
  var err error
  req := Request{
    Params: params,
  }
  var res Response
  path := fmt.Sprintf("/Blocks/%s/Setup", parentHash)
  err = s.PlainRequest(path, &req, &res)
  if err != nil { return "", err }
  if res.Error != "" {
    fmt.Printf("Error during setup:\n%s\n%s\n", res.Error, res.Details)
    return "", errors.New("error during setup")
  }
  return res.Hash, nil
}
