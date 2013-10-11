package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "log"
  "bufio"
  "path"
  //"strings"
  "errors"
  "strconv"
  "io/ioutil"
  "os"
  "time"
)


func worker(api HcsvlabApi,requests chan string,done chan int, annotationsProcessor chan *documentAnnotations) {
  for r := range requests {
    item, erro := api.GetItem(r)
    if erro != nil {
      log.Fatal(erro)
    }
    log.Println(item.Catalog_url)

    fileName := item.Metadata["Collection"] + item.Metadata["Identifier"]

    block := make(chan int,2)
    go func(item Item) {
      for _,doc := range item.Documents {
        data, err := api.Get(doc.Url)
        if err != nil {
          log.Fatal(err)
        }
        log.Println("Saving",fileName, "(",len(data),"bytes)")
        fo, err := os.Create(path.Join("data",fileName))
        if err != nil { panic(err) }
        // close fo on exit and check for its returned error
        defer func() {
            if err := fo.Close(); err != nil {
                panic(err)
            }
        }()
        w := bufio.NewWriter(fo)
        written, err := w.Write(data)
        if err != nil {
          log.Fatal(err)
        }
        log.Println(written, "bytes written to",fileName)
        w.Flush()
        block <- 1
      }
    }(item)

    go func(item Item) {
    /*  annotations, err := api.GetAnnotations(item)
      if err != nil {
        log.Fatal(err)
      }*/
      var da documentAnnotations
     // da.AnnotationList = &annotations.Annotations
      da.Filename = fileName
      annotationsProcessor <- &da
      block <-1
    }(item)

    <-block
    for i :=0 ; i < len(item.Documents); i++ {
      <-block
    }

    close(block)
  }

  done <- 1
}

type documentAnnotations struct {
  Filename string
  AnnotationList *AnnotationList
}


type ItemList struct {
  Name string
  Num_items float64
  Items []string
}

type DocIdentifier struct {
  Size string
  Url string
  Type string
}

type AnnotationList struct {
  Item_id string
  Item string
  Annotations_found float64
  Annotations []Annotation
}

type Annotation struct {
  Type string
  Label string
  Start float64
  End float64
}

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
  end := time.Now()
  log.Println("Time",url,end.Sub(start).Seconds(),resp.ContentLength)
  if err != nil {
    return
  }
  if resp.StatusCode != 200 {
    err = errors.New("Status " + strconv.Itoa(resp.StatusCode) + " from " + url)
    return
  }
    data, err = ioutil.ReadAll(resp.Body)
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

func main() {
  if len(os.Args) != 5 {
    fmt.Println(os.Args[0], ": Downloads and prints to standard out all the items associated with an itemlist in the HCSvLab API")
    fmt.Println("Usage:")
    fmt.Println("   ",os.Args[0], " <Number of workers> <API Base URL> <API Key> <item list id>")
    return
  }
  numWorkers,err := strconv.Atoi(os.Args[1])
  if err != nil {
    log.Fatal(err)
  }
  itemListId,err := strconv.Atoi(os.Args[4])
  if err != nil {
    log.Fatal(err)
  }
  log.Println("Number of workers:",numWorkers)
  api := HcsvlabApi{os.Args[2],os.Args[3]}

  requests := make(chan string,200)
  block := make(chan int,numWorkers)
  doneWriting := make(chan int,0)
  annotationsProcessor := make(chan *documentAnnotations,200)

  il, err := api.GetItemList(itemListId)
  if err != nil {
    log.Fatal(err)
  }

  for i := 0 ; i < numWorkers; i++ {
    go worker(api,requests,block,annotationsProcessor)
  }
  k := 0

  go func() {
    tagid := 1
    docid := 1
    for da := range annotationsProcessor {
   /*   for _, annotation := range da.AnnotationList.Annotations {
        log.Println(annotation)
        if int(annotation.End-annotation.Start) == 0 {
          fmt.Printf("%d\tannotation\t%d\t%s\t%d\t%d\t\t0\t\n",docid,tagid,annotation.Label,int(annotation.Start),int(annotation.End-annotation.Start))
        } else {
          fmt.Printf("%d\tTAG\t%d\t%s\t%d\t%d\t\t0\t\n",docid,tagid,annotation.Label,int(annotation.Start),int(annotation.End-annotation.Start))
        }
        tagid++
      }*/
       fmt.Println(da, tagid,docid)
    }
    doneWriting <- 1
  }()

  for _, s := range il.Items {
    requests <- s
    k++
  }
  log.Println("Number of items:",k)

  close(requests)


  select {
    case <-block:
      numWorkers--
      if numWorkers == 0 {
        close(annotationsProcessor)
        <-doneWriting
        return
      }
  }
}
