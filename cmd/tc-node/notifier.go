
package main

import (
  "fmt"
  "github.com/fatih/color"
  "github.com/k0kubun/go-ansi"
)

type Notifier struct {
  partial bool
  errorShown bool
}

func (n *Notifier) Partial(msg string) {
  ansi.EraseInLine(1)
  ansi.CursorHorizontalAbsolute(0)
  fmt.Print(msg)
  n.partial = true
}

func (n *Notifier) Final(msg string) {
  if n.partial {
    ansi.EraseInLine(1)
    ansi.CursorHorizontalAbsolute(0)
    fmt.Println(msg)
  }
  n.partial = false
}

func (n *Notifier) Warning(msg string) {
  if n.partial {
    ansi.EraseInLine(1)
    ansi.CursorHorizontalAbsolute(0)
    WarningFmt.Println(msg)
  }
  n.partial = false
}

func (n *Notifier) Error(err error) {
  if n.partial {
    DangerFmt.Println(" failed")
    n.partial = false
  }
  if err.Error() == "API error" {
    if remote.LastError != "" {
      DangerFmt.Println(remote.LastError)
    }
    if remote.LastDetails != "" {
      fmt.Println(remote.LastDetails)
    }
  } else {
    DangerFmt.Println(err.Error())
  }
  n.errorShown = true
}

var SuccessFmt = color.New(color.Bold, color.FgGreen)
var DangerFmt = color.New(color.Bold, color.FgRed)
var WarningFmt = color.New(color.Bold, color.FgYellow)
var ImportantFmt = color.New(color.Bold, color.FgHiWhite)
var NoticeFmt = color.New(color.FgHiBlack)
var GameKeyFmt = color.New(color.Bold, color.FgWhite)
