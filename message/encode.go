
package message

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "strings"
  "github.com/golang-collections/collections/stack"
  "github.com/pkg/errors"
)

type Encoder struct {
  dec *json.Decoder
  out *bytes.Buffer
  nesting *stack.Stack
  firstItem bool
  depth int
  signature *string
}

func (r *Encoder) Value() (err error) {
  var t json.Token
  t, err = r.Token()
  if err != nil { return }
  switch v := t.(type) {
  case json.Delim: // [ ] { }
    switch v {
    case '{':
      r.Object()
      r.Token()
    case '[':
      r.Array()
      r.Token()
    default:
      return errors.Errorf("unexpected token %v", v)
    }
  case string:
    fmt.Fprintf(r.out, "%q", v)
  case float64:
    fmt.Fprintf(r.out, "%v", v)
  default:
    if v == nil {
      fmt.Fprint(r.out, "null")
    } else {
      fmt.Fprintf(r.out, "%v", v)
    }
  }
  return nil
}

func (r *Encoder) Token() (res interface{}, err error) {
  res, err = r.dec.Token()
  if err == io.EOF {
    res = r.nesting.Pop()
    if res == nil { return }
    err = nil
  } else if err != nil {
    return
  }
  switch v := res.(type) {
  case rune:
    switch v {
    case '}':
      r.CloseObject()
    case ']':
      r.CloseArray()
    default:
      return nil, errors.New("bad delimiter on stack")
    }
  case json.Delim: // [ ] { }
    switch v {
    case '{':
      r.depth++
      r.nesting.Push('}')
    case '}':
      r.CloseObject()
    case '[':
      r.depth++
      r.nesting.Push(']')
    case ']':
      r.CloseArray()
    default:
      return nil, errors.New("bad delimiter")
    }
  }
  return
}

func (r *Encoder) Array() (err error) {
  fmt.Fprintf(r.out, "[\n")
  r.firstItem = true
  for r.dec.More() {
    r.Item()
    if err = r.Value(); err != nil { return }
  }
  return nil
}

func (r *Encoder) CloseArray() {
  r.depth--
  if (r.firstItem) {
    r.firstItem = false;
  } else {
    fmt.Fprintf(r.out, "\n")
    fmt.Fprint(r.out, strings.Repeat("  ", r.depth))
  }
  fmt.Fprintf(r.out, "]")
}

func (r *Encoder) Object() (err error) {
  fmt.Fprintf(r.out, "{\n")
  r.firstItem = true
  for r.dec.More() {
    r.Item()
    if err = r.Key(); err != nil { return }
    if err = r.Value(); err != nil { return }
  }
  return nil
}

func (r *Encoder) CloseObject() {
  r.depth--
  if (r.firstItem) {
    r.firstItem = false;
  } else {
    fmt.Fprintf(r.out, "\n")
    fmt.Fprint(r.out, strings.Repeat("  ", r.depth))
  }
  fmt.Fprintf(r.out, "}")
}

func (r *Encoder) Item() {
  if (r.firstItem) {
    r.firstItem = false
  } else {
    fmt.Fprint(r.out, ",\n")
  }
  fmt.Fprint(r.out, strings.Repeat("  ", r.depth))
}

func (r *Encoder) Key() error {
  t, err := r.Token()
  if err != nil {
    return err
  }
  switch v := t.(type) {
  case string:
    fmt.Fprintf(r.out, "%q: ", v)
    return nil
  default:
    return errors.Errorf("expected key, got %v", v)
  }
}

func (r *Encoder) Bytes() []byte {
  return r.out.Bytes()
}

func Encode(b []byte) ([]byte, error) {
  r := Encoder{
    dec: json.NewDecoder(bytes.NewReader(b)),
    out: new(bytes.Buffer),
    nesting: stack.New(),
    firstItem: false,
    depth: 0,
    signature: nil,
  }
  err := r.Value()
  if err != nil {
    return []byte{}, err
  }
  return r.Bytes(), nil
}

func InjectSignature(b []byte, sig string) []byte {
  out := new(bytes.Buffer)
  l := len(b)
  out.Write(b[:l - 2])
  fmt.Fprintf(out, ",\n  %q: %q\n}", "signature", sig)
  return out.Bytes()
}
