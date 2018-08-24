
package api

import (
  "bytes"
  "encoding/json"
  "errors"
  "io"
  "net/http"
  "tezos-contests.izibi.com/tezos-play/keypair"
  "tezos-contests.izibi.com/tezos-play/message"
)

type Server struct {
  Base string
}

func New (base string) (*Server) {
  return &Server{
    Base: base,
  }
}

func (s *Server) request(path string, body io.Reader, result interface{}) (err error) {
  var resp *http.Response
  if body == nil {
    resp, err = http.Get(s.Base + path)
  } else {
    resp, err = http.Post(s.Base + path,
      "application/json; charset=utf-8", body)
  }
  if err != nil { return }
  if resp.StatusCode < 200 || resp.StatusCode >= 299 {
    err = errors.New(resp.Status)
    return
  }
  if resp.StatusCode == 200 {
    err = json.NewDecoder(resp.Body).Decode(&result)
  }
  return
}

func (s *Server) PlainRequest(path string, body interface{}, result interface{}) (err error) {
  var b *bytes.Buffer
  if body != nil {
    b = new(bytes.Buffer)
    json.NewEncoder(b).Encode(body)
  }
  return s.request(path, b, result)
}

func (s *Server) SignedRequest(keyPair *keypair.KeyPair, path string, msg interface{}, result interface{}) (err error) {
  var b []byte
  b, err = message.Sign(keyPair, msg)
  if err != nil { return }
  err = s.request(path, bytes.NewReader(b), result)
  return
}
