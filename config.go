package main

import (
  "encoding/json"
  "io/ioutil"
)

type ConfigPaths struct {
  QueryAll string
  IndriBuildIndex string
  IndriRunQuery string
}

type Config struct {
  Binaries ConfigPaths
  ApiPath string
  WebDir string
}

func ReadConfig() (conf Config, err error) {
  bytes, err := ioutil.ReadFile("config.json")
  if err != nil {
    return
  }
  err = json.Unmarshal(bytes,&conf)
  return
}
