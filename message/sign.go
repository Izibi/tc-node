
package message

import (
  "bytes"
  "encoding/base64"
  "encoding/json"
  "crypto/hmac"
  "crypto/sha512"
  "golang.org/x/crypto/ed25519"
  "tezos-contests.izibi.com/task1-game/keypair"
)

const GAME_API_KEY = "z4fRNQW1xJidCuGO0l0G4eR97bkwPSdTbXSyMzeCRes="

func Sign(keys *keypair.KeyPair, obj interface{}) ([]byte, error) {
  var err error
  var res []byte
  plain := new(bytes.Buffer)
  json.NewEncoder(plain).Encode(obj)
  var encoded []byte
  encoded, err = Encode(plain.Bytes())
  if err != nil {
    return res, err
  }
  var rawPriv []byte
  rawPriv, err = keys.RawPrivate()
  if err != nil {
    return res, err
  }
  gameApiKey, _ :=  base64.StdEncoding.DecodeString(GAME_API_KEY)
  hasher := hmac.New(sha512.New, []byte(gameApiKey))
  hasher.Write([]byte(encoded))
  hash := hasher.Sum(nil)[:32]
  rawSig := ed25519.Sign(rawPriv, hash)
  encSig := base64.StdEncoding.EncodeToString(rawSig) + ".sig.ed25519"
  res = InjectSignature(encoded, encSig)
  return res, nil
}

