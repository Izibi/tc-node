
package api

type NewProtocolRequest struct {
  Interface string `json:"interface"`
  Implementation string `json:"implementation"`
}
type NewProtocolResponse struct {
  Hash string `json:"hash"`
}

func (s *Server) NewProtocol(intf string, impl string) (string, error) {
  var err error
  var res NewProtocolResponse
  err = s.PlainRequest("/protocols", NewProtocolRequest{
    Interface: intf,
    Implementation: impl,
  }, &res)
  if err != nil { return "", err }
  return res.Hash, nil
}
