
package block_store

import (
  "archive/zip"
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "github.com/go-errors/errors"
)

type Block struct {
  Type string `json:"type"`
  Parent string `json:"parent"`
  Round uint32 `json:"round"`
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
  blockDir := filepath.Join(s.BlockDir, hash)
  err = os.MkdirAll(blockDir, os.ModePerm)
  if err != nil { err = errors.Errorf("failed to create '%s'", blockDir); return }
  err = s.fetch(hash, blockDir)
  if err != nil { return }
  var b []byte
  blockPath := filepath.Join(blockDir, "block.json")
  b, err = ioutil.ReadFile(blockPath)
  if err != nil { err = errors.Errorf("failed to read '%s'", blockPath); return }
  block := new(Block)
  err = json.Unmarshal(b, block)
  if err != nil { err = errors.Errorf("bad block '%s': %s", hash, err); return }
  s.hashMap[hash] = block
  seqDir := filepath.Join(s.BlockDir, strconv.FormatUint(uint64(block.Round), 10))
  err = os.RemoveAll(seqDir)
  if err != nil { err = errors.Errorf("failed to remove directory '%s'", seqDir); return }
  os.Rename(blockDir, seqDir)
  if err != nil { err = errors.Errorf("failed to move block from '%s' to '%s'", blockDir, seqDir); return }
  res = block
  return
}

func (s *Store) GetChain(firstBlock string, lastBlock string) (err error) {
  var block *Block
  hash := lastBlock
  for hash != "" {
    block, err = s.Get(hash)
    if err != nil { return err }
    if firstBlock == hash { break }
    hash = block.Parent
  }
  return
}

func (s *Store) fetch(hash string, dest string) (err error) {

  destPrefix := filepath.Clean(dest) + string(os.PathSeparator)

  var resp *http.Response
  zipUrl := fmt.Sprintf("%s/%s/zip", s.BaseUrl, hash)
  resp, err = http.Get(zipUrl)
  if err != nil {
    err = errors.Errorf("failed to GET %s: %s", zipUrl, err)
    return
  }
  if resp.StatusCode != 200 {
    err = errors.Errorf("failed to GET %s: %s", zipUrl, resp.Status)
    return
  }

  bs, err := ioutil.ReadAll(resp.Body)
  if err != nil { err = errors.Wrap(err, 0); return }
  r, err := zip.NewReader(bytes.NewReader(bs), int64(len(bs)))
  if err != nil { err = errors.Wrap(err, 0); return }

  for _, f := range r.File {
    fpath := filepath.Join(dest, f.Name)
    if !strings.HasPrefix(fpath, destPrefix) {
      err = errors.Errorf("illegal file path %s", fpath)
      return
    }
    if f.FileInfo().IsDir() {
      os.MkdirAll(fpath, os.ModePerm)
    } else {
      err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
      if err != nil { err = errors.Wrap(err, 0); return }
      var outFile *os.File
      outFile, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
      if err != nil { err = errors.Wrap(err, 0); return }
      var inFile io.ReadCloser
      inFile, err = f.Open()
      if err != nil {
        outFile.Close()
        err = errors.Wrap(err, 0)
        return
      }
      _, err = io.Copy(outFile, inFile)
      outFile.Close()
      inFile.Close();
      if err != nil { err = errors.Wrap(err, 0); return }
    }
  }

  return
}
