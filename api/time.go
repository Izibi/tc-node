
package api

import (
  "time"
)

func (s *Server) GetTime() (res time.Time, err error) {
  var timeStr string
  err = s.GetRequest("/Time", &timeStr)
  if err != nil { return }
  t, err := time.Parse(time.RFC3339, timeStr)
  if err != nil { return }
  return t, nil
}
