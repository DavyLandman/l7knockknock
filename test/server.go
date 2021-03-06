package main

import (
    "flag"
    "bufio"
    "net"
    "fmt"
    "os"
    "io"
    "time"
    "bytes"
    "strings"
    "math/rand"
    "github.com/cespare/xxhash"
    "encoding/hex"
    "strconv"
    "log"
    "runtime"
    "runtime/pprof"
    "os/signal"
    "syscall"
)


func main() {
    port := flag.Int("port", 4000, "Port to accept connections on.")
    specialPort := flag.Int("specialPort", 4001, "Port to echo hello on.")
    cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
    flag.Parse()

    if *cpuprofile != "" {
        f, err := os.Create(*cpuprofile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }

    sig_terms := make(chan os.Signal, 2)
    signal.Notify(sig_terms, os.Interrupt, syscall.SIGTERM)

    l, err := net.Listen("tcp", ":" + strconv.Itoa(*port))
    if err != nil {
        fmt.Println("ERROR", err)
        os.Exit(1)
    }

    l2, err := net.Listen("tcp", ":" + strconv.Itoa(*specialPort))
    if err != nil {
        fmt.Println("ERROR", err)
        os.Exit(1)
    }

    println("Filling random data")
    random_bytes := make([]byte, 1024*1024)
    _, err = io.ReadFull(rand.New(rand.NewSource(time.Now().UnixNano())), random_bytes)
    if err != nil {
        fmt.Println("ERROR", err)
        os.Exit(1)
    }
    println("Done, ready to serve")

    runtime.GOMAXPROCS(runtime.NumCPU() / 2)


    go func() {
        for  {
            conn, err := l.Accept()
            if err != nil {
                fmt.Println("ERROR", err)
                continue
            }
            go clientConnection(conn, random_bytes)
        }
    }() // main function that handles new connections


    go func() {
        for  {
            conn, err := l2.Accept()
            if err != nil {
                fmt.Println("ERROR", err)
                continue
            }
            conn.Write([]byte("HELLO"))
            conn.Close()
        }
    }()

    <-sig_terms // on sigterm, quit running (helps with profiling information, and correct error reporting)
}

func Min(x, y int) int {
    if x < y {
        return x
    }
    return y
}

func clientConnection(conn net.Conn, random_bytes []byte) {
    defer conn.Close()

    r := bufio.NewReader(conn)
    line, err := r.ReadString(byte('\n'))
    if err != nil {
        fmt.Println("ERROR", err)
        return
    }
    line = strings.TrimSuffix(line, "\n")
    required_size, err := strconv.Atoi(line)
    if err != nil {
        fmt.Println("ERROR", err)
        return
    }

    hasher := xxhash.New()
    send := 0
    for send < required_size {
        written, err := conn.Write(random_bytes[:Min(required_size - send, len(random_bytes))])
        
        if err != nil {
            fmt.Println("ERROR", err)
            return
        }
        hasher.Write(random_bytes[:written])
        send += written
    }

    expected_hash := hex.EncodeToString(hasher.Sum(nil))
    
    line, err = r.ReadString(byte('\n'))
    if err != nil {
        fmt.Println("ERROR", err)
        return
    }
    line = strings.TrimSuffix(line, "\n")


    result := []byte{'F', 'A', 'I', 'L'}
    if line == expected_hash {
        result = []byte{'O', 'K', 'O', 'K'} 
    }

    _, err = io.CopyN(conn, bytes.NewBuffer(result), 4)
    if err != nil {
        fmt.Println("ERROR", err)
        return
    }
    conn.Close()
}
