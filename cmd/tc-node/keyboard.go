
package main

import (
  "github.com/eiannone/keyboard"
)

type Keypress struct {
  ch rune
  key keyboard.Key
}

func keyboardChannel() (<-chan Keypress /*, func ()*/) {
  kch := make(chan Keypress)
  go func() {
    var err error
    err = keyboard.Open()
    if err == nil {
      for {
        ch, key, err := keyboard.GetKey()
        if err != nil { break }
        if ch == 0 && key == 0 { break /* closed */ }
        kch<- Keypress{ch, key}
      }
    }
    keyboard.Close()
    close(kch)
    return
  }()
  /* var kcl := func() { keyboard.Close() } */
  return kch /* , kcl */
}
