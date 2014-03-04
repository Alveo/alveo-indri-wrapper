package main

import (
  "fmt"
  "log"
  "bufio"
  "os"
  "path"
  "errors"
  "net/http"
  "encoding/json"
  "bytes"
  "os/exec"
  "strconv"
  "code.google.com/p/gorest"
  "github.com/TimothyJones/hcsvlabapi"
)

type ErrorResponse struct {
  Class string `json:"type"`
  Err string `json:"error"`
}

type MatchItem struct {
  DocId string `json:"docid"`
  Location int64 `json:"location"`
  Match string `json:"match"`
}

type AllQueryResult struct {
  Class string `json:"type"`
  Matches []*MatchItem
}



func stringError(err error) (string) {
  var response = ErrorResponse{"error",err.Error()}
  result, errMars := json.Marshal(response);
  if errMars != nil {
    return "{type: \"error\",message: \"Cannot marshal json error\"}"
  }
  return string(result)
}



func worker(api hcsvlabapi.Api,requests chan string,done chan int, annotationsProcessor chan *documentAnnotations) {
  for r := range requests {
    item, erro := api.GetItemFromUri(r)
    if erro != nil {
      log.Println("Worker encountered",erro)
      continue
    }
    log.Println(item.Catalog_url)

    fileName := item.Metadata["dc:identifier"]

    block := make(chan int,2)
    go func(item hcsvlabapi.Item) {
      data, err := api.Get(item.Primary_text_url)
      if err != nil {
        log.Println("Error obtaining item from API",err)
        block <- 1
        return
      }
      log.Println("Saving",fileName, "(",len(data),"bytes)")
      fo, err := os.Create(path.Join("data",fileName))
      if err != nil {
        log.Println("Error opening file for item",err)
        block <- 1
        return
      }
      // close fo on exit and check for its returned error
      defer func() {
        if err := fo.Close(); err != nil {
            log.Println("Worker couldn't close the item's file",err)
        }
        log.Println("Finished",fileName)
      }()
      w := bufio.NewWriter(fo)
      written, err := w.Write(data)
      if err != nil {
        log.Println("Error writing file for item",err)
        block <- 1
        return
      }
      log.Println(written, "bytes written to",fileName)
      w.Flush()
      block <- 1
    }(item)

    go func(item hcsvlabapi.Item) {
      annotations, err := api.GetAnnotations(item)
      if err != nil {
        log.Println("Error obtaining annotations",err)
        block <- 1
        return
      }
      da := &documentAnnotations{fileName,&annotations}
      annotationsProcessor <- da
      block <-1
    }(item)

    <-block
    <-block
    log.Println("Moving on from",fileName)

    close(block)
  }

  done <- 1
}

type documentAnnotations struct {
  Filename string
  AnnotationList* hcsvlabapi.AnnotationList
}

//Service Definition
type IndriService struct {
  gorest.RestService `root:"/"`
  query  gorest.EndPoint `method:"GET" path:"/query/docs/{itemList:int}/{query:string}" output:"string"`
  queryall  gorest.EndPoint `method:"GET" path:"/query/all/{itemList:int}/{query:string}" output:"string"`
  index    gorest.EndPoint `method:"GET" path:"/index/{itemList:int}" output:"string"`
}

func(serv IndriService) Queryall(itemList int, query string) string{
  log.Println("Query all recieved request for itemlist",itemList, " with query",query)
  cmd := exec.Command("/Users/tim/office/c/snipped/example", path.Join("repos",strconv.FormatInt(int64(itemList),10)),query)
  out := bytes.NewBuffer(nil)
  cmd.Stdout = out
  err := cmd.Run()
  if err != nil {
    log.Println("QueryAll encountered this error:",err)
    return stringError(err)
  }

  // read from the string from the buffer, becasue the out buffer contains no EOF
  scanner := bufio.NewScanner(bytes.NewBufferString(out.String()))

  state := 1

  var location int64
  location = 0
  docId := ""
  match := ""

  var res AllQueryResult

  res.Class = "result"
  res.Matches = make([]*MatchItem, 0, 1000)

  for scanner.Scan() {
    // 1st docid
    // 2nd position
    // 3rd match
    if state == 1 {
      docId = scanner.Text()
      state = 2
    } else if state == 2 {
      location, err = strconv.ParseInt(scanner.Text(),10,64)
      if err != nil {
        log.Println("Couldn't parse location in result")
      }
      state = 3
    } else if state == 3 {
      match = scanner.Text()
      item := &MatchItem{docId,location,match}
      res.Matches = append(res.Matches,item)
      log.Println("Match complete",item)

      location = 0
      docId = ""
      match = ""
      state = 1
    }
  }
  if err := scanner.Err(); err != nil {
    return stringError(err)
  }
  result, errMars := json.Marshal(res);
  if errMars != nil {
    return "{type: \"error\",message: \"Cannot marshal json response\"}"
  }
  return string(result)
}

func(serv IndriService) Query(itemList int, query string) string{
  cmd := exec.Command("/Users/tim/indri-5.6/runquery/IndriRunQuery", "-index=" + path.Join("repos",strconv.FormatInt(int64(itemList),10)),"-query="+query,"-count=1000")
  var out bytes.Buffer
  cmd.Stdout = &out
  err := cmd.Run()
  if err != nil {
    log.Println("Query encountered this error:",err)
    return stringError(err)
  }
  serv.ResponseBuilder().SetContentType("text/plain; charset=\"utf-8\"")
  return out.String()
}

func(serv IndriService) Index(itemList int) string{
  // Declare upfront because of use of goto
  cmd := exec.Command("/Users/tim/indri-5.6/buildindex/IndriBuildIndex", "index.properties")
  var out bytes.Buffer

  // processing begins here
  err := obtainAndIndex(10,itemList,"http://ic2-hcsvlab-staging2-vm.intersect.org.au/","ApysuCqJPV4zxYSpqaej")
  if err != nil {
    goto errHandle
  }
  log.Println("Beginning indexing")
  cmd.Stdout = &out
  err = cmd.Run()
  if err != nil {
    goto errHandle
  }
  serv.ResponseBuilder().SetContentType("text/plain; charset=\"utf-8\"")
  return out.String()
  
  errHandle:

  log.Println("Index encountered this error:",err)
  return stringError(err)
}

func main() {
  gorest.RegisterService(new(IndriService)) //Register our service
  http.Handle("/",gorest.Handle())
  http.ListenAndServe(":8787",nil)
}

func obtainAndIndex(numWorkers int, itemListId int,apiBase string, apiKey string) (err error){
  log.Println("Number of workers:",numWorkers)
  api := hcsvlabapi.Api{apiBase,apiKey}
  ver,err := api.GetVersion()
  if err != nil {
    return
  }
  if ver.Api_version != "Sprint_20_demo" {
    err = errors.New("Server API version is incorrect:" + ver.Api_version)
    return
  }

  requests := make(chan string,200)
  block := make(chan int,numWorkers)
  doneWriting := make(chan int,0)
  annotationsProcessor := make(chan *documentAnnotations,200)

  il, err := api.GetItemList(itemListId)
  if err != nil {
    return
  }

  for i := 0 ; i < numWorkers; i++ {
    go worker(api,requests,block,annotationsProcessor)
  }
  k := 0

  go func() {
    // This is the annotations processor
    // It also writes the index file
    tagid := 1
    docid := 1
    log.Println("Starting to annotate")
    defer func() {
      doneWriting <- 1
    }()

    // Create annotations writer
    annFo, err := os.Create("annotation.offsets")
    if err != nil {
      log.Println("Error unable to create annotations offset file",err)
      return
    }
    annWriter := bufio.NewWriter(annFo)

    defer func() {
      annWriter.Flush()
      if err := annFo.Close(); err != nil {
        log.Println("Error unable to close annotations offset file",err)
      }
      log.Println("Closing annFo")
    }()

    // Create index properties writer
    ixFo, err := os.Create("index.properties")
    if err != nil {
      log.Println("Error unable to create index description file",err)
      return
    }
    ixWriter := bufio.NewWriter(ixFo)

    defer func() {
      log.Println("Closing ixFo")
      ixWriter.Flush()
      if err := ixFo.Close(); err != nil {
        log.Println("Couldn't close the ixWriter",err)
      }
    }()

    fmt.Fprintf(ixWriter,"<parameters>\n<index>%s</index>\n",path.Join("repos",strconv.FormatInt(int64(itemListId),10)))
    fmt.Fprintf(ixWriter,"<corpus>\n")
    fmt.Fprintf(ixWriter,"  <class>xml</class>\n")
    fmt.Fprintf(ixWriter,"  <annotations>annotation.offsets</annotations>\n")
    fmt.Fprintf(ixWriter,"  <path>data</path>\n")

    for da := range annotationsProcessor {
      log.Println("writing annotations for",da.Filename)

      if da.AnnotationList != nil {
        for _, annotation := range da.AnnotationList.Annotations {
          aEnd,err := strconv.Atoi(annotation.End)
          if err != nil {
            log.Println("Unable to convert end annotation",annotation.End,"to int")
            continue
          }
          aStart,err := strconv.Atoi(annotation.Start)
          if err != nil {
            log.Println("Unable to convert end annotation",annotation.Start,"to int")
            continue
          }
          if aEnd-aStart == 0 {
            fmt.Fprintf(annWriter,"%s\tannotation\t%d\t%s\t%d\t%d\t\t0\t\n",da.Filename,tagid,annotation.Label,aStart,aEnd-aStart)
          } else {
            fmt.Fprintf(annWriter,"%s\tTAG\t%d\t%s\t%d\t%d\t\t0\t\n",da.Filename,docid,tagid,annotation.Label,aStart,aEnd-aStart)
          }
          tagid++
        }
      }
      docid++
    }
    fmt.Fprintf(ixWriter,"</corpus>\n</parameters>")
    log.Println("Finished ix descriptor")
  }()

  for _, s := range il.Items {
    requests <- s
    k++
  }
  log.Println("Number of items:",k)

  close(requests)

  for {
    select {
      case <-block:
       numWorkers--
       log.Println("A thread done: ",numWorkers, " remaining")
       if numWorkers == 0 {
         close(annotationsProcessor)
         <-doneWriting
         return
        }
    }
  }
}
