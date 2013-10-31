package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "log"
  "errors"
  "strconv"
  "io/ioutil"
  "time"
)



// A representation of an item list from the HCSvLab API
type ItemList struct {
  Name string
  Num_items float64
  Items []string
}

// A representation of a document within an item from the HCSvLab API
type DocIdentifier struct {
  Size string
  Url string
  Type string
}

// A representation of the annotations associated with an item from the HCSvLab API
type AnnotationList struct {
  Item_id string
  Item string
  Annotations_found float64
  Annotations []Annotation
}

// An annotation associated with a documents. AnnotationLists have more than one of these.
type Annotation struct {
  Type string
  Label string
  Start float64
  End float64
}

// An item that contains metadata about a document from the HCSvLab API
type Item struct {
 Catalog_url string
 Metadata map[string]string
 Primary_text_url string
 Annotations_url string
 Documents []DocIdentifier
}

type HcsvlabApi struct {
  base string
  key string
}


func (api *HcsvlabApi) Get(url string) (data []byte, err error) {
  client := &http.Client{}
  req, err := http.NewRequest("GET", url, nil)
  req.Header.Add("X-API-KEY",api.key)
  log.Println("Requesting ",url,"with key",api.key)
  start := time.Now()
  resp, err := client.Do(req)
  if err != nil {
    return
  }
  if resp.StatusCode != 200 {
    err = errors.New("Status " + strconv.Itoa(resp.StatusCode) + " from " + url)
    return
  }
  data, err = ioutil.ReadAll(resp.Body)
  end := time.Now()
  log.Println("Time",url,end.Sub(start).Seconds(),resp.ContentLength)
  resp.Body.Close()
  return
}

func (api *HcsvlabApi) GetItemList(list int) (il ItemList, err error)  {
  url := fmt.Sprintf("%s/item_lists/%d.json",api.base, list)
  data, err := api.Get(url);
  if err != nil {
    return
  }
  err = json.Unmarshal(data,&il)
  return
}

func (api *HcsvlabApi) GetAnnotations(item Item) (al AnnotationList, err error)  {
  data, err := api.Get(item.Annotations_url)
  if err != nil {
    return
  }
  err = json.Unmarshal(data,&al)
  return
}

func (api *HcsvlabApi) GetItem(url string)  (item Item, err error) {
  data, err := api.Get(url)
  if err != nil {
    return
  }
  err = json.Unmarshal(data,&item);
  return
}
