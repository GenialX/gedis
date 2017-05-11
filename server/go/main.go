// Copyright 2017 By GenialX. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
    "bytes"
    "fmt"
    "io"
    "net"
    "strings"
    "time"
)

const (
    SERVER_NETWORK = "tcp"
    SERVER_ADDRESS = "127.0.0.1:1721"
    DELIMITER      = '\n'
    MAX_THREAD     = 20
)

var ch = make(chan int, MAX_THREAD)
var m = make(map[string]string)

func read(conn net.Conn) (string, error) {
    readBytes := make([]byte, 1)
    var buffer bytes.Buffer
    for {
        _, err := conn.Read(readBytes)
        if err != nil {
            return "", err
        }
        readByte := readBytes[0]
        if readByte == DELIMITER {
            break
        }
        buffer.WriteByte(readByte)
    }
    return buffer.String(), nil
}

func write(conn net.Conn, content string) (int, error) {
    var buffer bytes.Buffer
    buffer.WriteString(content)
    buffer.WriteByte(DELIMITER)
    return conn.Write(buffer.Bytes())
}

func printLog(role string, sn int, format string, args ...interface{}) {
    if !strings.HasSuffix(format, "\n") {
        format += "\n"
    }
    fmt.Printf("%s[%d]: %s", role, sn, fmt.Sprintf(format, args...))
}

func printServerLog(format string, args ...interface{}) {
    printLog("Server", 0, format, args...)
}

func handleConn(conn net.Conn) {
    defer func() {
        conn.Close()
        <-ch
    }()
    for {
        conn.SetReadDeadline(time.Now().Add(120 * time.Second))
        reqStr, err := read(conn)
        if err != nil {
            if err == io.EOF {
                printServerLog("The connection is closed by another side.")
            } else {
                printServerLog("Read Error: %s", err)
            }
            break
        }
        printServerLog("Received request msg: %s", reqStr)
        cmdBytes := []byte(reqStr)
        reqLen   := len(reqStr) - 1
        p1 := bytes.IndexByte(cmdBytes, ' ')
        if (p1 > -1) {
            cmd := make([]byte, p1)
            _ = copy(cmd, cmdBytes[:p1])
            switch {
            case string(cmd) == "get":
                key := make([]byte, reqLen - p1 - 1)
                _ = copy(key, cmdBytes[p1 + 1:])
                value := m[string(key)]
                printServerLog("The value of the key: %s", value)
                _, err = write(conn, value)
                if err != nil {
                    printServerLog("Write Error: %s", err)
                }
            case string(cmd) == "set":
                p2 := bytes.IndexByte(cmdBytes[p1 + 1:], ' ')
                key := make([]byte, p2)
                _ = copy(key, cmdBytes[p1 + 1:p1 + p2 + 1])
                value := make([]byte, reqLen - p1 - p2 - 1)
                _ = copy(value, cmdBytes[p1 + p2 + 2:])
                m[string(key)] = string(value)
                _, err = write(conn, "true")
                if err != nil {
                    printServerLog("Write Error: %s", err)
                }
            default:
            }
        }
    }
}

func serverGo() {
    var listener net.Listener
    listener, err := net.Listen(SERVER_NETWORK, SERVER_ADDRESS)
    if err != nil {
        printServerLog("Listen Error: %s", err)
        return
    }
    defer listener.Close()
    printServerLog("Listening...(local address: %s)", listener.Addr())
    for {
        conn, err := listener.Accept() // block until one client request
        if err != nil {
            printServerLog("Accept Error: %s", err)
        }
        printServerLog("Established one connection with one client. (remote address: %s)", conn.RemoteAddr())
        ch <- 1
        go handleConn(conn)
    }
}

func main() {
    serverGo()
}
