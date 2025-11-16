package main

import (
    "flag"
    "fmt"
    "time"
    "github.com/quantarax/backend/daemon/manager"
)

func main() {
    path := flag.String("db", "cas.db", "Path to Bolt CAS DB")
    maxAge := flag.Duration("max-age", 24*time.Hour, "Max age for CAS entries")
    flag.Parse()

    cas, err := manager.OpenBoltCAS(*path)
    if err != nil { panic(err) }
    defer cas.Close()
    removed, err := cas.GC(*maxAge)
    if err != nil { panic(err) }
    fmt.Printf("CAS GC removed %d entries older than %s\n", removed, maxAge.String())
}
