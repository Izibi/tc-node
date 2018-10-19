// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Copyright 2015 Andrew Stuart
// Copyright 2018 Sebastien Carlier
// Use of this source code is governed by The MIT License (MIT).

package sse

import (
  "fmt"
  "bufio"
  "bytes"
  "github.com/go-errors/errors"
  "io"
  "net/http"
  "strconv"
  "time"
)

type client struct {
  uri string
  retryDelay time.Duration
  ch chan<- string
}

func Connect(uri string) (<-chan string, error) {
  ch := make(chan string)
  c := &client{uri, 3 * time.Second, ch}
  r, err := c.connect("")
  if err != nil { return nil, err }
  go c.readEventSource(r)
  return ch, nil
}

func (c *client) connect(lastId string) (io.ReadCloser, error) {
  // fmt.Printf("SSE client connecting to %s\n", c.uri)
  req, err := http.NewRequest("GET", c.uri, nil)
  if err != nil {
    // fmt.Printf("SSE client request failed: %v\n", err)
    return nil, err
  }
  req.Header.Set("Accept", "text/event-stream")
  if lastId != "" {
    req.Header.Set("Last-Event-ID", lastId)
  }
  res, err := http.DefaultClient.Do(req)
  if err != nil {
    fmt.Printf("SSE client GET failed: %v\n", err)
    return nil, err
  }
  if res.StatusCode != 200 {
    return nil, errors.Errorf("SSE server error (%s)", res.Status)
  }
  return res.Body, nil
}

func (c *client) readEventSource(r io.ReadCloser) {
  defer r.Close()
  defer close(c.ch)

  type Line struct {
    line []byte
    err error
  }
  lines := make(chan Line)
  defer close(lines)
  readLines := func (r io.ReadCloser) {
    br := bufio.NewReader(r)
    for {
      line, err := br.ReadBytes('\n')
      lines <- Line{line, err}
      if err != nil {
        break
      }
    }
  }
  go readLines(r)
  var reconnect <-chan time.Time
  /*  Enable to force a periodic reconnect:
  var reconnectTicker *time.Ticker
  reconnectTicker = time.NewTicker(5 * time.Second)
  reconnect = reconnectTicker.C
  */

  var err error
  var lastId string
  var eventType string
  var dataBuffer *bytes.Buffer = new(bytes.Buffer)
  for {
    var l Line
    select {
      case l = <-lines:
        break
      case <-reconnect:
        // reconnectTicker.Stop()
        r.Close()
        continue
    }
    if l.err != nil {
      for {
        time.Sleep(c.retryDelay)
        r, err = c.connect(lastId)
        if err == nil { break }
      }
      go readLines(r)
      eventType = ""
      dataBuffer.Reset()
      continue
    }
    line := l.line
    /* Reset (and enable) the exponential backoff. */
    if len(line) < 2 {
      // For Web browsers, the appropriate steps to dispatch the event are as follows:
      // Set the last event ID string of the event source to the value of the last event ID buffer. The buffer does not get reset, so the last event ID string of the event source remains set to this value until the next time it is set by the server.
      // If the data buffer is an empty string, set the data buffer and the event type buffer to the empty string and return.
      data := dataBuffer.Bytes()
      if len(data) == 0 {
        eventType = ""
        continue
      }
      // If the data buffer's last character is a U+000A LINE FEED (LF) character, then remove the last character from the data buffer.
      if data[len(data)-1] == '\n' {
        data = data[0:len(data)-1]
      }
      // Let event be the result of creating an event using MessageEvent, in the
      // relevant Realm of the EventSource object. Initialize event's type
      // attribute to "message", its data attribute to data, its origin attribute
      // to the serialization of the origin of the event stream's final URL
      // (i.e., the URL after redirects), and its lastEventId attribute to the
      // last event ID string of the event source.
      type_ := "message"
      // If the event type buffer has a value other than the empty string, change the type of the newly created event to equal the value of the event type buffer.
      if eventType != "" {
        type_ = eventType
      }
      // Set the data buffer and the event type buffer to the empty string.
      dataBuffer.Reset()
      eventType = ""
      // Queue a task which, if the readyState attribute is set to a value other than CLOSED, dispatches the newly created event at the EventSource object.
      if type_ == "message" {
        c.ch<- string(data)
      }
      continue
    }
    if line[0] == byte(':') {
      // If the line starts with a U+003A COLON character (:), ignore the line.
      continue
    }

    var field, value []byte
    colonIndex := bytes.IndexRune(line, ':')
    if colonIndex != -1 {
      // If the line contains a U+003A COLON character character (:)
      // Collect the characters on the line before the first U+003A COLON character (:),
      // and let field be that string.
      field = line[:colonIndex]
      // Collect the characters on the line after the first U+003A COLON character (:),
      // and let value be that string.
      value = line[colonIndex+1:len(line)-1]
      // If value starts with a single U+0020 SPACE character, remove it from value.
      if len(value) > 0 && value[0] == ' ' {
        value = value[1:]
      }
    } else {
      // Otherwise, the string is not empty but does not contain a U+003A COLON character character (:)
      // Use the whole line as the field name, and the empty string as the field value.
      field = line
      value = []byte{}
    }
    // The steps to process the field given a field name and a field value depend on the field name,
    // as given in the following list. Field names must be compared literally,
    // with no case folding performed.
    switch string(field) {
    case "event":
      // Set the event type buffer to field value.
      eventType = string(value)
    case "id":
      // If the field value does not contain U+0000 NULL, then set the last event ID buffer to the field value.
      // Otherwise, ignore the field.
      if len(value) > 0 {
        lastId = string(value)
      }
    case "retry":
      // If the field value consists of only ASCII digits, then interpret the
      // field value as an integer in base ten, and set the event stream's
      // reconnection time to that integer. Otherwise, ignore the field.
      var retry int64
      retry, err = strconv.ParseInt(string(value), 10, 64)
      if err == nil && retry > 0 {
        c.retryDelay = time.Duration(retry) * time.Millisecond
      }
    case "data":
      // Append the field value to the data buffer,
      dataBuffer.Write(value)
      // then append a single U+000A LINE FEED (LF) character to the data buffer.
      dataBuffer.WriteString("\n")
    default:
      //Otherwise. The field is ignored.
      continue
    }
  }

}
