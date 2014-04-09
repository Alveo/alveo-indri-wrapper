package main

import (
  "fmt"
  "log"
  "net/http"
  "code.google.com/p/gorest"
)

var (
 config Config
)

func main() {
  var err error
  config, err = ReadConfig()
  if err != nil {
    fmt.Println("Unable to read config file, not starting.")
    fmt.Println("Error:",err)
    log.Println("Error:",err)
    return
  }
  log.Println("Progress: Server starting with",config)
  initialiseLocks()
  gorest.RegisterMarshaller("application/x-www-form-urlencoded", NewUrlMarshaller())
  gorest.RegisterService(new(IndriService)) //Register our service
  http.Handle("/",gorest.Handle())
  http.ListenAndServe(":8787",nil)
}

