
package api

type NewProtocolRequest struct {
  LibraryInterface string `json:"library_interface"`
  LibraryImplementation string `json:"library_implementation"`
}
type NewProtocolResponse struct {
  Hash string `json:"hash"`
}

func (s *Server) NewProtocol(intf string, impl string) (string, error) {
  var err error
  var res NewProtocolResponse
  err = s.PlainRequest("/protocols", NewProtocolRequest{
    LibraryInterface: intf,
    LibraryImplementation: impl,
  }, &res)
  if err != nil { return "", err }
  return res.Hash, nil
}
