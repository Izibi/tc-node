
package api

type GetTimeResponse struct {
  ServerTime string `json:"server_time"`
}

func (s *Server) GetTime() (res string, err error) {
  resp := GetTimeResponse{}
  err = s.GetRequest("/Time", &resp)
  if err != nil { return }
  return resp.ServerTime, nil
}
