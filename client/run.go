
package client

import (
  "bytes"
  "fmt"
  "os"
  "os/exec"
  "runtime"
  "strings"
)

type CommandEnv struct {
  RoundNumber uint64
  PlayerNumber uint32
  NbCycles uint
  BotId uint32
}

func runCommand(shellCmd string, env CommandEnv) (string, error) {
  var cmd *exec.Cmd
  if runtime.GOOS == "windows" {
    cmd = exec.Command("cmd.exe", "/C", shellCmd)
  } else {
    cmd = exec.Command("sh", "-c", shellCmd)
  }
  cmd.Env = append(os.Environ(),
    fmt.Sprintf("ROUND_NUMBER=%d", env.RoundNumber),
    fmt.Sprintf("PLAYER_NUMBER=%d", env.PlayerNumber),
    fmt.Sprintf("NB_CYCLES=%d", env.NbCycles),
    fmt.Sprintf("BOT_ID=%d", env.BotId),
  )
  input := fmt.Sprintf("%d %d", env.RoundNumber, env.PlayerNumber)
  cmd.Stdin = strings.NewReader(input)
  cmd.Stderr = os.Stderr
  var out bytes.Buffer
  cmd.Stdout = &out
  err := cmd.Run()
  if err != nil { return "", err }
  return out.String(), nil
}
