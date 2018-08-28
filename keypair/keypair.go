
package keypair

import (
  "crypto/rand"
  "encoding/base64"
  "encoding/json"
  "os"
  "strings"
  "golang.org/x/crypto/ed25519"
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

func New () (res *KeyPair, err error) {
  pub, pri, err := ed25519.GenerateKey(rand.Reader)
  if err != nil { return nil, err }
  res = &KeyPair{
    Curve: "ed25519",
    Public: base64.StdEncoding.EncodeToString(pub) + ".ed25519",
    Private: base64.StdEncoding.EncodeToString(pri) + ".ed25519",
  }
  return res, nil
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

func (kp *KeyPair) Write (filename string) error {
  file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
  if err != nil { return err }
  defer file.Close()
  err = json.NewEncoder(file).Encode(kp)
  if err != nil { return err }
  return nil
}
