
package client

import (
  "bytes"
  "fmt"
  "os"
  "os/exec"
  "strings"
)

type CommandEnv struct {
  RoundNumber uint64
  PlayerNumber uint32
}

func runCommand(shellCmd string, env CommandEnv) (string, error) {
  cmd := exec.Command("sh", "-c", shellCmd)
  cmd.Env = append(os.Environ(),
    fmt.Sprintf("ROUND_NUMBER=%d", env.RoundNumber),
    fmt.Sprintf("PLAYER_NUMBER=%d", env.PlayerNumber),
  )
  input := fmt.Sprintf("%d %d", env.RoundNumber, env.PlayerNumber)
  cmd.Stdin = strings.NewReader(input)
  var out bytes.Buffer
  cmd.Stdout = &out
  err := cmd.Run()
  if err != nil { return "", err }
  return out.String(), nil
}