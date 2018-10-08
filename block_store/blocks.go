
package block_store

import (
  "archive/zip"
  "bytes"
  "crypto/sha1"
  "encoding/base64"
  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "github.com/fatih/color"
  "github.com/go-errors/errors"
  "github.com/json-iterator/go"
)

var noticeFmt = color.New(color.FgHiBlack)

type Block struct {
  Type string `json:"type"`
  Parent string `json:"parent"`
  Sequence uint32 `json:"sequence"`
}

type Store struct {
  BaseUrl string
  BlocksDir string
  blockByHash map[string]*Block
  Index *Index
}

func New (baseUrl string, blocksDir string) (*Store) {
  store := &Store{}
  store.BaseUrl = baseUrl
  store.BlocksDir = blocksDir
  store.Index = NewIndex(blocksDir)
  _ = store.loadHashes()
  _ = store.Index.Load()
  return store
}

func (s *Store) Clear() error {
  var err error
  s.blockByHash = map[string]*Block{}
  err = os.RemoveAll(s.BlocksDir)
  if err != nil { return err }
  err = os.MkdirAll(s.BlocksDir, os.ModePerm)
  if err != nil { return err }
  return nil
}

func (st *Store) Get(hash string) (res *Block, err error) {
  /* A block that has been loaded has been verified, so just return it. */
  res = st.blockByHash[hash]
  if res != nil { return res, nil }
  /* If the hash is in the index, name the blockDir with the round number.
     Otherwise, name the blockdir with the hash (maybe renamed later). */
  var blockDir string
  var round uint64
  var ok bool
  round, ok = st.Index.GetRoundByHash(hash)
  if ok {
    blockDir = filepath.Join(st.BlocksDir, strconv.FormatUint(round, 10))
  } else {
    blockDir = filepath.Join(st.BlocksDir, hash)
  }
  /* Create the blockDir, fetch and unzip the block. */
  err = os.MkdirAll(blockDir, os.ModePerm)
  if err != nil { err = errors.Errorf("failed to create '%s'", blockDir); return }
  err = st.fetch(hash, blockDir)
  if err != nil { return }
  /* Load and hash block.json, verify it matches our expectation. */
  var bs []byte
  blockPath := filepath.Join(blockDir, "block.json")
  bs, err = ioutil.ReadFile(blockPath)
  if err != nil { err = errors.Errorf("failed to read '%s'", blockPath); return }
  var computedHash = hashBlock(bs)
  if hash != computedHash {
    return nil, errors.Errorf("block %s has bad hash %s", hash, computedHash)
  }
  /* Parse the block and load it into the cache. */
  block := new(Block)
  err = json.Unmarshal(bs, block)
  if err != nil { err = errors.Errorf("bad block '%s': %s", hash, err); return }
  st.blockByHash[hash] = block
  /* Attempt to read a round number. */
  round, err = st.readRoundNumber(hash)
  if err == nil {
    /* Move the block to a round-named directory, and write the associtation in
       the index. */
    shortDir := filepath.Join(st.BlocksDir, strconv.FormatUint(round, 10))
    err = os.RemoveAll(shortDir)
    if err != nil { err = errors.Errorf("failed to remove directory '%s'", shortDir); return }
    os.Rename(blockDir, shortDir)
    if err != nil { err = errors.Errorf("failed to move block from '%s' to '%s'", blockDir, shortDir); return }
    err = st.Index.Add(hash, round)
    if err != nil { return }
  } else {
    /* Block has no round number, leave it with a hash name. */
    err = nil
  }
  res = block
  return
}

func (s *Store) GetChain(firstBlock string, lastBlock string) (err error) {
  var block *Block
  /* scan block store, read and hash all block.json files, store hash -> sequence number map */
  hash := lastBlock
  for hash != "" {
    block, err = s.Get(hash)
    if err != nil { return err }
    if firstBlock == hash { break }
    hash = block.Parent
  }
  return
}

func (s *Store) readRoundNumber(hash string) (uint64, error) {
  statePath := filepath.Join(s.BlocksDir, hash, "state.json")
  b, err := ioutil.ReadFile(statePath)
  if err != nil { return 0, errors.Wrap(err, 0) }
  return jsoniter.Get(b, "round").ToUint64(), nil
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

func hashBlock(bs []byte) string {
  hasher := sha1.New()
  hasher.Write(bs)
  return base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
}


func (st *Store) loadHashes() error {
  st.blockByHash = make(map[string]*Block)
  filepath.Walk(st.BlocksDir, func (path string, info os.FileInfo, err error) error {
    if err != nil { return err }
    if !info.IsDir() { return nil }
    if path == st.BlocksDir { return nil }
    blockPath := filepath.Join(path, "block.json")
    blockBytes, err := ioutil.ReadFile(blockPath)
    if err != nil {
      return errors.Errorf("failed to read '%s': %v", blockPath, err)
    }
    hash := hashBlock(blockBytes)
    var block Block
    err = json.Unmarshal(blockBytes, &block)
    if err != nil {
      /* Bad block, delete to force redownload. */
      _ = os.RemoveAll(path)
      return nil
    }
    st.blockByHash[hash] = &block
    return nil
  })
  /* fail silently */
  return nil
}
