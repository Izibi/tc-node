
package api

import (
  "bytes"
  "encoding/json"
  "github.com/go-errors/errors"
  "fmt"
  "io"
  "net/http"
  "os"
  "tezos-contests.izibi.com/backend/signing"
)

type Server struct {
  Base string
  ApiKey string
  teamKeyPair *signing.KeyPair
  client *http.Client
  LastError string /* last error */
  LastDetails string /* details of last error */
}

type ServerResponse struct {
  Result interface{} `json:"result"`
  Error string `json:"error"`
  Details string `json:"details"`
}

func New (base string, apiKey string, teamKeyPair *signing.KeyPair) (*Server) {
  return &Server{
    Base: base,
    ApiKey: apiKey,
    teamKeyPair: teamKeyPair,
    client: new(http.Client),
  }
}

func (s *Server) Author() string {
  return "@" + s.teamKeyPair.Public
}

func (s *Server) GetRequest(path string, result interface{}) (err error) {
  var req *http.Request
  req, err = http.NewRequest("GET", s.Base + path, nil)
  if err != nil { err = errors.Wrap(err, 0); return }
  req.Header.Add("X-API-Version", Version)
  var resp *http.Response
  resp, err = s.client.Do(req)
  if err != nil { err = errors.Wrap(err, 0); return }
  if resp.StatusCode < 200 || resp.StatusCode >= 299 {
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    err = errors.Errorf("Failed to GET %s: %v\n%s", s.Base + path, err, buf.String())
    return
  }
  sr := ServerResponse{result, "", ""}
  err = json.NewDecoder(resp.Body).Decode(&sr)
  if sr.Error != "" {
    s.LastError = sr.Error
    s.LastDetails = sr.Details
    return errors.New("API error")
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

func (s *Server) PlainRequest(path string, msg interface{}, result interface{}) error {
  b := new(bytes.Buffer)
  err := json.NewEncoder(b).Encode(msg)
  if err != nil { return err }
  return s.postRequest(path, b, result)
}

func (s *Server) SignedRequest(path string, msg interface{}, result interface{}) error {
  if s.teamKeyPair == nil {
    return errors.Errorf("team keypair is missing")
  }
  b := new(bytes.Buffer)
  err := json.NewEncoder(b).Encode(msg)
  if err != nil { return errors.Errorf("malformed message in request: %s", err) }
  bs, err := signing.Sign(s.teamKeyPair.Private, s.ApiKey, b.Bytes())
  if err != nil { return errors.Errorf("failed to sign message: %s", err) }
  resp := ServerResponse{result, "", ""}
  err = s.postRequest(path, bytes.NewReader(bs), &resp)
  if err != nil { return errors.Errorf("failed to contact API: %s", err) }
  if resp.Error != "" {
    s.LastError = resp.Error
    s.LastDetails = resp.Details
    return errors.New("API error")
  }
  return nil
}
