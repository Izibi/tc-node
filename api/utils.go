
package api

import (
  "bytes"
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "net/http"
  "os"
  "tezos-contests.izibi.com/game/keypair"
  "tezos-contests.izibi.com/game/message"
)

type Server struct {
  Base string
  ApiKey string
  client *http.Client
}

func New (base string, apiKey string) (*Server) {
  return &Server{
    Base: base,
    ApiKey: apiKey,
    client: new(http.Client),
  }
}

func (s *Server) GetRequest(path string, result interface{}) (err error) {
  var req *http.Request
  req, err = http.NewRequest("GET", s.Base + path, nil)
  if err != nil { return }
  req.Header.Add("X-API-Version", Version)
  var resp *http.Response
  resp, err = s.client.Do(req)
  if err != nil { return }
  if resp.StatusCode < 200 || resp.StatusCode >= 299 {
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    fmt.Fprintln(os.Stderr, buf.String())
    err = errors.New(resp.Status)
    return
  }
  if resp.StatusCode == 200 {
    err = json.NewDecoder(resp.Body).Decode(&result)
  }
  return
}

func (s *Server) postRequest(path string, body io.Reader, result interface{}) (err error) {
  var resp *http.Response
  resp, err = http.Post(s.Base + path,
    "application/json; charset=utf-8", body)
  if err != nil { return }
  if resp.StatusCode < 200 || resp.StatusCode >= 299 {
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    fmt.Fprintln(os.Stderr, buf.String())
    err = errors.New(resp.Status)
    return
  }
  if resp.StatusCode == 200 {
    err = json.NewDecoder(resp.Body).Decode(&result)
  }
  return
}

func (s *Server) PlainRequest(path string, body interface{}, result interface{}) (err error) {
  b := new(bytes.Buffer)
  json.NewEncoder(b).Encode(body)
  return s.postRequest(path, b, result)
}

func (s *Server) SignedRequest(keyPair *keypair.KeyPair, path string, msg interface{}, result interface{}) (err error) {
  var b []byte
  b, err = message.Sign(keyPair, s.ApiKey, msg)
  if err != nil { return }
  err = s.postRequest(path, bytes.NewReader(b), result)
  return
}
