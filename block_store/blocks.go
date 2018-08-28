
package block_store

import (
  "archive/zip"
  "bytes"
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
)

type Block struct {
  Type string `json:"type"`
  Parent string `json:"parent"`
  Sequence uint32 `json:"sequence"`
}

type Store struct {
  BaseUrl string
  BlockDir string
  hashMap map[string]*Block
}

func New (baseUrl string, cacheDir string) (*Store) {
  return &Store{
    BaseUrl: baseUrl,
    BlockDir: cacheDir,
    hashMap: map[string]*Block{},
  }
}

func (s *Store) Clear() (err error) {
  err = os.RemoveAll(s.BlockDir)
  if err != nil { return }
  err = os.MkdirAll(s.BlockDir, os.ModePerm)
  if err != nil { return }
  return
}

func (s *Store) Get(hash string) (res *Block, err error) {
  res = s.hashMap[hash]
  err = nil
  if res != nil { return }
  blockDir := filepath.Join(s.BlockDir, hash)
  err = os.MkdirAll(blockDir, os.ModePerm)
  if err != nil { return }
  err = s.fetch(hash, blockDir)
  if err != nil { return }
  var b []byte
  b, err = ioutil.ReadFile(filepath.Join(blockDir, "block.json"))
  if err != nil { return }
  block := new(Block)
  err = json.Unmarshal(b, block)
  if err != nil { return }
  s.hashMap[hash] = block
  seqDir := filepath.Join(s.BlockDir, strconv.FormatUint(uint64(block.Sequence), 10))
  err = os.RemoveAll(seqDir)
  if err != nil { return }
  os.Rename(blockDir, seqDir)
  if err != nil { return }
  res = block
  return
}

func (s *Store) GetChain(hash string) (err error) {
  var block *Block
  for true {
    block, err = s.Get(hash)
    if err != nil { return err }
    hash = block.Parent
    if hash == "" { break }
  }
  return
}

func (s *Store) fetch(hash string, dest string) (err error) {

  destPrefix := filepath.Clean(dest) + string(os.PathSeparator)

  var resp *http.Response
  resp, err = http.Get(s.BaseUrl + "/" + hash + ".zip")
  if err != nil { return }
  if resp.StatusCode != 200 {
    err = errors.New(resp.Status)
    return
  }

  bs, err := ioutil.ReadAll(resp.Body)
  if err != nil { return }
  r, err := zip.NewReader(bytes.NewReader(bs), int64(len(bs)))
  if err != nil { return }

  for _, f := range r.File {
    fpath := filepath.Join(dest, f.Name)
    if !strings.HasPrefix(fpath, destPrefix) {
      err = fmt.Errorf("illegal file path %s", fpath)
      return
    }
    if f.FileInfo().IsDir() {
      os.MkdirAll(fpath, os.ModePerm)
    } else {
      err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
      if err != nil { return }
      var outFile *os.File
      outFile, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
      if err != nil { return }
      var inFile io.ReadCloser
      inFile, err = f.Open()
      if err != nil { outFile.Close(); return }
      _, err = io.Copy(outFile, inFile)
      outFile.Close()
      inFile.Close();
      if err != nil { return }
    }
  }

  /* Load sequence from block.json */

  return
}
