
package keypair

import (
  "encoding/base64"
  "encoding/json"
  "os"
  "strings"
)

type KeyPair struct {
  Curve string `json:"curve"`
  Public string `json:"public"`
  Private string `json:"private"`
}

func (s *KeyPair) RawPrivate() ([]byte, error) {
  b64 := strings.Split(s.Private, ".")[0]
  return base64.StdEncoding.DecodeString(b64)
}

func Read (filename string) (*KeyPair, error) {
  var err error
  file, err := os.Open(filename)
  if err != nil {
    return nil, err
  }
  defer file.Close()
  var res KeyPair
  err = json.NewDecoder(file).Decode(&res)
  if err != nil {
    return nil, err
  }
  return &res, nil
}
