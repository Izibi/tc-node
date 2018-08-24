
package api

type TaskParams struct {
  NbTeams uint32 `json:"nb_teams"`
  MapSide uint32 `json:"map_side"`
}

type NewChainRequest struct {
  TaskParams TaskParams `json:"task_params"`
  LibraryInterface string `json:"library_interface"`
  LibraryImplementation string `json:"library_implementation"`
}
type NewChainResponse struct {
  Hash string `json:"hash"`
}

func (s *Server) NewChain(params TaskParams, intf string, impl string) (string, error) {
  var err error
  var res NewChainResponse
  err = s.PlainRequest("/chains", NewChainRequest{
    TaskParams: params,
    LibraryInterface: intf,
    LibraryImplementation: impl,
  }, &res)
  if err != nil { return "", err }
  return res.Hash, nil
}
