package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "log"
  "errors"
  "strconv"
  "io/ioutil"
  "os"
)


func (api *HcsvlabApi) process(url string) (result string,err error){
  client := &http.Client{}
  req, err := http.NewRequest("GET", url, nil)
  req.Header.Add("X-API-KEY",api.key)
  resp, err := client.Do(req)
  if err != nil {
    return
  }
  if resp.StatusCode != 200 {
    err = errors.New("Status " + strconv.Itoa(resp.StatusCode))
  }
  data, err := ioutil.ReadAll(resp.Body)
  resp.Body.Close()
  if err != nil {
    return
  }
  result = string(data)
  return
}


func worker(api HcsvlabApi,requests chan string,done chan int) {
  for r := range requests {
    data, err := api.process(r)
    if err != nil {
      log.Fatal(err)
    }
    fmt.Println(data)
  }
  done <- 1
}


type ItemList struct {
  Name string
  Num_items float64
  Items []string
}

type HcsvlabApi struct {
  base string
  key string
}

func (api *HcsvlabApi) GetItemList(list int) (il ItemList, err error)  {
  client := &http.Client{}
  url := fmt.Sprintf("%s/item_lists/%d.json",api.base, list)

  req, err := http.NewRequest("GET", url, nil)
  req.Header.Add("X-API-KEY",api.key)
  resp, err := client.Do(req)
  if err != nil {
    return
  }
  if resp.StatusCode != 200 {
    err = errors.New("Status " + strconv.Itoa(resp.StatusCode))
  }
  data, err := ioutil.ReadAll(resp.Body)
  resp.Body.Close()
  if err != nil {
    return
  }
  err = json.Unmarshal(data,&il)
  return
}

func main() {
  if len(os.Args) != 4 {
    fmt.Println(os.Args[0], ": Downloads and prints to standard out all the items associated with an itemlist in the HCSvLab API")
    fmt.Println("Usage:")
    fmt.Println("   ",os.Args[0], " <Number of workers> <API Base URL> <API Key>")
    return
  }
  numWorkers,err := strconv.Atoi(os.Args[1])
  if err != nil {
    log.Fatal(err)
  }
  log.Println("Number of workers:",numWorkers)
  api := HcsvlabApi{os.Args[2],os.Args[3]}

  requests := make(chan string,200)
  block := make(chan int,numWorkers)

  il, err := api.GetItemList(11)
  if err != nil {
    log.Fatal(err)
  }

  for i := 0 ; i < numWorkers; i++ {
    go worker(api,requests,block)
  }
  k := 0
  for _, s := range il.Items {
    requests <- s
    k++
  }
  log.Println("Number of items:",k)

  close(requests)
  for i := 0 ; i < numWorkers; i++ {
    <-block
  }
}
