
package client

type IdleChannel struct {
  input  chan Command
  output chan Command
  buffer Command
}

func NewIdleChannel() *IdleChannel {
  ch := &IdleChannel{
    input:  make(chan Command),
    output: make(chan Command),
  }
  go ch.run()
  return ch
}

func (ch *IdleChannel) In() chan<- Command {
  return ch.input
}

func (ch *IdleChannel) Out() <-chan Command {
  return ch.output
}

func (ch *IdleChannel) Close() {
  close(ch.input)
}

func (ch *IdleChannel) run() {
  var input = ch.input
  var output chan Command
  var open, full bool
  var next Command
  for input != nil || output != nil {
    select {
    case output <- next:
      full = false
    default:
      select {
      case next, open = <-input:
        if open {
          full = true
        } else {
          input = nil
        }
      case output <- next:
        full = false
      }
    }
    if full {
      output = ch.output
    } else {
      output = nil
    }
  }
  close(ch.output)
}
