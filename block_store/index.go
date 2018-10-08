/*
  Maintain an 'index.txt' file in the block store directory.
  Each line of this file has the following format:

  hash round

*/

package block_store

import (
  "io/ioutil"
  "os"
  "path/filepath"
  "fmt"
  "strconv"
  "strings"
)

type Index struct {
  path string
  roundByHash map[string]uint64
}

func NewIndex(blocksDir string) *Index {
  var idx = Index{}
  idx.path = filepath.Join(blocksDir, "index.txt")
  return &idx
}

func (idx *Index) Load() error {
  var err error
  idx.roundByHash = map[string]uint64{}
  var bs []byte
  bs, err = ioutil.ReadFile(idx.path)
  if err != nil {
    if os.IsNotExist(err) { return nil }
    return err
  }
  var lines = strings.Split(string(bs), "\n")
  var line string
  for _, line = range lines {
    var parts = strings.Split(line, " ")
    if len(parts) != 2 { continue }
    var hash = parts[0]
    var round uint64
    round, err = strconv.ParseUint(parts[1], 10, 32)
    idx.roundByHash[hash] = uint64(round)
  }
  return nil
}

func (idx *Index) Add(hash string, round uint64) (err error) {
  var f *os.File
  f, err = os.OpenFile(idx.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
  if err != nil { return }
  defer f.Close()
  var line = fmt.Sprintf("%s %d\n", hash, round)
  _, err = f.WriteString(line)
  if err != nil { return }
  idx.roundByHash[hash] = round
  return nil
}

func (idx *Index) GetRoundByHash(hash string) (uint64, bool) {
  val, ok := idx.roundByHash[hash]
  return val, ok
}
