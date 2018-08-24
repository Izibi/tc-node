
package block_store

import (
  "archive/zip"
  "bytes"
  "errors"
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path"
  "strings"
)

type Store struct {
  BaseUrl string
  CacheDir string
  cacheMap map[string]interface{}
}

type Block struct {

}

func New (baseUrl string, cacheDir string) (*Store) {
  return &Store{
    BaseUrl: baseUrl,
    CacheDir: cacheDir,
  }
}

func (s *Store) Get(hash string) (res interface{}, err error) {
  res = s.cacheMap[hash]
  err = nil
  if res != nil { return }
  blockDir := path.Join(s.CacheDir, hash)
  err = os.MkdirAll(blockDir, os.ModePerm)
  if res != nil { return }
  err = s.Fetch(hash, blockDir)
  // TODO: load blockDir/block.json
  return
}

func (s *Store) Fetch(hash string, dest string) (err error) {

  destPrefix := path.Clean(dest) + string(os.PathSeparator)

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
    fpath := path.Join(dest, f.Name)
    if !strings.HasPrefix(fpath, destPrefix) {
      err = fmt.Errorf("illegal file path %s", fpath)
      return
    }
    if f.FileInfo().IsDir() {
      os.MkdirAll(fpath, os.ModePerm)
    } else {
      err = os.MkdirAll(path.Dir(fpath), os.ModePerm)
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

  return
}
