package main

import (
  "fmt"
  "log"
  "bufio"
  "time"
  "os"
  "path"
  "errors"
  "strings"
  "sync"
  "net/http"
  "encoding/json"
  "bytes"
  "os/exec"
  "strconv"
  "code.google.com/p/gorest"
  "github.com/TimothyJones/hcsvlabapi"
)

var (
 itemListsInProgress map[int]int
 itemListSize map[int]int
 errorsFromIndex map[int]error // there can only be one error per index request, so that's ok.
 progressMutex sync.Mutex
 config Config
)

func getUrlForDocId(docId string) string {
  return config.ApiPath + "/catalog/" + docId
}


func worker(api hcsvlabapi.Api,requests chan string,done chan int, annotationsProcessor chan *documentAnnotations,itemListHelper *ItemListHelper) {
  for r := range requests {
    item, erro := api.GetItemFromUri(r)
    if erro != nil {
      log.Println("Error: Worker encountered",erro)
      continue
    }
    fileName := item.Metadata["hcsvlab:handle"]

    block := make(chan int,2)
    go func(item hcsvlabapi.Item) {
      data, err := api.Get(item.Primary_text_url)
      if err != nil {
        log.Println("Error: obtaining item from API",err)
        block <- 1
        return
      }
      log.Println("Progress: Saving",fileName, "(",len(data),"bytes)")
      fo, err := os.Create(path.Join(itemListHelper.DataLocation(),fileName))
      if err != nil {
        log.Println("Error: opening file for item",err)
        block <- 1
        return
      }
      // close fo on exit and check for its returned error
      defer func() {
        if err := fo.Close(); err != nil {
          log.Println("Error: Worker couldn't close the item's file",err)
        }
      }()
      w := bufio.NewWriter(fo)
      _, err = w.Write(data)
      if err != nil {
        log.Println("Error: writing file for item",err)
        block <- 1
        return
      }
      w.Flush()
      block <- 1
    }(item)

    go func(item hcsvlabapi.Item) {
      annotations, err := api.GetAnnotations(item)
      if err != nil {
        log.Println("Error: obtaining annotations",err)
        block <- 1
        return
      }
      da := &documentAnnotations{path.Join(itemListHelper.DataLocation(),fileName),&annotations}
      annotationsProcessor <- da
      block <-1
    }(item)

    <-block
    <-block

    progressMutex.Lock()
    itemListsInProgress[itemListHelper.Id]++
    progressMutex.Unlock()

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
  query  gorest.EndPoint `method:"GET" path:"/query/doc/{itemList:int}/{query:string}" output:"string"`
  queryall  gorest.EndPoint `method:"GET" path:"/query/all/{itemList:int}/{query:string}" output:"string"`
  index    gorest.EndPoint `method:"GET" path:"/index/{itemList:int}" output:"string"`
  progress gorest.EndPoint `method:"GET" path:"/progress/{itemList:int}/{after:string}" output:"string"`
}

func(serv IndriService) Progress(itemList int,after string) string{
  log.Println("Info: Index progress requested for itemlist",itemList)
  itemListHelper := &ItemListHelper{itemList}
  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")

  progressMutex.Lock()
  numProcessed := itemListsInProgress[itemList]
  err := errorsFromIndex[itemList]
  progressMutex.Unlock()

  if err != nil {
    return stringError(err)
  }

  // Ignore the error, because it means there's just no index yet
  createdTime, _ := itemListHelper.CreatedTime()

  completed := false
  if createdTime != "" {
    timeAfter, err := time.Parse(TimeFormat, after)
    if err != nil {
      return stringError(err)
    }
    timeCreatedTime, err := time.Parse(TimeFormat,createdTime)

    completed = timeAfter.Before(timeCreatedTime)
  }

  res := IndexProgressResponse{"progress",numProcessed,itemListSize[itemList],completed,createdTime}

  result, errMars := json.Marshal(res);
  if errMars != nil {
    return stringError(errMars)
  }
  return string(result)
}

func(serv IndriService) Queryall(itemList int, query string) string{
  log.Println("Info: Query all recieved request for itemlist",itemList, " with query",query)
  itemListHelper := &ItemListHelper{itemList}
  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")

  if strings.TrimSpace(query) == "" {
    return stringError(errors.New("Empty query"))
  }

  indexCreatedTime, err := itemListHelper.CreatedTime()
  if err != nil {
    return stringError(err)
  }

  cmd := exec.Command(config.Binaries.QueryAll, itemListHelper.RepoLocation(),query)
  out := bytes.NewBuffer(nil)
  cmd.Stdout = out
  err = cmd.Run()
  if err != nil {
    log.Println("Error: QueryAll encountered this error:",err)
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

  res.Class = "result-all"
  res.Matches = make([]*MatchItem, 0, 1000)
  res.IndexCreatedTime = indexCreatedTime

  for scanner.Scan() {
    // 1st docid
    // 2nd position
    // 3rd match
    if state == 1 {
      docId = itemListHelper.docIdForFile(scanner.Text())
      state = 2
    } else if state == 2 {
      location, err = strconv.ParseInt(scanner.Text(),10,64)
      if err != nil {
        log.Println("Error: Couldn't parse location in result")
      }
      state = 3
    } else if state == 3 {
      match = scanner.Text()
      item := &MatchItem{docId,getUrlForDocId(docId),location,match}
      res.Matches = append(res.Matches,item)
      log.Println("Progress: Found match",item)

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
    return stringError(errMars)
  }
  return string(result)
}

func(serv IndriService) Query(itemList int, query string) string{
  log.Println("Info: Query for doc matches received:",query)
  itemListHelper := &ItemListHelper{itemList}
  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")

  indexCreatedTime, err := itemListHelper.CreatedTime()
  if err != nil {
    return stringError(err)
  }

  cmd := exec.Command(config.Binaries.IndriRunQuery, "-index=" + itemListHelper.RepoLocation(),"-query="+query,"-count=1000")
  var out bytes.Buffer
  cmd.Stdout = &out
  err = cmd.Run()
  if err != nil {
    log.Println("Error: Query encountered this error:",err)
    return stringError(err)
  }
  scanner := bufio.NewScanner(bytes.NewBufferString(out.String()))

  var res DocQueryResult

  res.Class = "result-doc"
  res.Matches = make([]*MatchDoc, 0, 1000)
  res.IndexCreatedTime = indexCreatedTime

  for scanner.Scan() {
    A := strings.Split(scanner.Text(),"\t")
    if len(A) != 4 {
      log.Println("Error: response contains less than four fields")
    } else {
      start, err := strconv.ParseInt(A[2],10,64)
      if err != nil {
        log.Println("Error: Couldn't parse start in result")
      }
      end, err := strconv.ParseInt(A[3],10,64)
      if err != nil {
        log.Println("Error: Couldn't parse end in result")
      }
      docId := itemListHelper.docIdForFile(A[1])
      match := &MatchDoc{docId,getUrlForDocId(docId),start,end}
      res.Matches = append(res.Matches,match)
    }
  }
  result, errMars := json.Marshal(res);
  if errMars != nil {
    return stringError(errMars)
  }
  str := string(result)
  str = str[strings.LastIndex(str,"\n"):]
  return str
}

func(serv IndriService) Index(itemList int) string{
  log.Println("Info: Request to index itemList",itemList)
  itemListHelper := &ItemListHelper{itemList}
  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")
  // Declare upfront because of use of goto
  cmd := exec.Command(config.Binaries.IndriBuildIndex, path.Join(itemListHelper.ConfigLocation(),"index.properties"))
  var out bytes.Buffer
  var err error

  progressMutex.Lock()
  if itemListsInProgress[itemList] != 0 {
    log.Println("Error: Indexing already in progress")
    err = errors.New("Itemlist is already being indexed. Please wait for the indexing to complete")
    progressMutex.Unlock()
    return stringError(err)
  }
  itemListsInProgress[itemList] = 1
  delete(errorsFromIndex,itemList)
  progressMutex.Unlock()

  go func() {
    defer func() {
      progressMutex.Lock()
      itemListsInProgress[itemList] = 0
      progressMutex.Unlock()
    }()

    // processing begins here
    err = obtainAndIndex(10,itemList,config.ApiPath,config.ApiKey)
    if err != nil {
      goto errHandle
    }

    log.Println("Progress: Removing old index")
    err = itemListHelper.RemoveRepo()
    if err != nil {
      goto errHandle
    }

    log.Println("Progress: Beginning indexing")
    cmd.Stdout = &out
    err = cmd.Run()
    if err != nil {
      goto errHandle
    }
    log.Println("Progress: Removing data")
    err = itemListHelper.RemoveData()
    if err != nil {
      goto errHandle
    }
    log.Println("Progress: Indexing complete")

    return

    errHandle:

    log.Println("Error: Index encountered this error:",err)

    progressMutex.Lock()
    errorsFromIndex[itemList] = err
    progressMutex.Unlock()
    return
  }()

  res := &IndexResponse{"indexing",time.Now().Format(TimeFormat)}

  result, errMars := json.Marshal(res);
  if errMars != nil {
    return stringError(errMars)
  }
  return string(result)
}

func main() {
  var err error
  config, err = ReadConfig()
  if err != nil {
    fmt.Println("Unable to read config file, not starting.")
    fmt.Println("Error:",err)
    return
  }
  fmt.Println(config)
  gorest.RegisterService(new(IndriService)) //Register our service
  itemListsInProgress = make(map[int]int)
  itemListSize = make(map[int]int)
  errorsFromIndex = make(map[int]error)
  http.Handle("/",gorest.Handle())
  http.ListenAndServe(":8787",nil)
}

func obtainAndIndex(numWorkers int, itemListId int,apiBase string, apiKey string) (err error){
  log.Println("Progress: Checking itemlists to see if",itemListId, "is in progress")
  log.Println("Progress: Indexing itemlist",itemListId,"with number of workers:",numWorkers)
  api := hcsvlabapi.Api{apiBase,apiKey}
  ver,err := api.GetVersion()
  if err != nil {
    return
  }

  if ver.Api_version != "Sprint_22_demo" {
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

  itemListHelper := &ItemListHelper{itemListId}

  err = itemListHelper.MakeReadyForDownload()
  if err != nil {
    return
  }

  for i := 0 ; i < numWorkers; i++ {
    go worker(api,requests,block,annotationsProcessor,itemListHelper)
  }
  k := 0

  go func() {
    // This is the annotations processor
    // It also writes the index file
    tagid := 1
    docid := 1
    defer func() {
      doneWriting <- 1
    }()

    // Create annotations writer
    annFo, err := os.Create(path.Join(itemListHelper.ConfigLocation(),"annotation.offsets"))
    if err != nil {
      log.Println("Error: unable to create annotations offset file",err)
      return
    }
    annWriter := bufio.NewWriter(annFo)

    defer func() {
      annWriter.Flush()
      if err := annFo.Close(); err != nil {
        log.Println("Error: unable to close annotations offset file",err)
      }
    }()

    // Create index properties writer
    ixFo, err := os.Create(path.Join(itemListHelper.ConfigLocation(),"index.properties"))
    if err != nil {
      log.Println("Error: unable to create index description file",err)
      return
    }
    ixWriter := bufio.NewWriter(ixFo)

    defer func() {
      ixWriter.Flush()
      if err := ixFo.Close(); err != nil {
        log.Println("Error: Couldn't close the ixWriter",err)
      }
    }()

    fmt.Fprintf(ixWriter,"<parameters>\n<index>%s</index>\n",itemListHelper.RepoLocation())
    fmt.Fprintf(ixWriter,"<corpus>\n")
    fmt.Fprintf(ixWriter,"  <class>xml</class>\n")
    fmt.Fprintf(ixWriter,"  <annotations>%s</annotations>\n",path.Join(itemListHelper.ConfigLocation(),"annotation.offsets"))
    fmt.Fprintf(ixWriter,"  <path>%s</path>\n",itemListHelper.DataLocation())

    for da := range annotationsProcessor {
      log.Println("Progress: Writing annotations for",da.Filename)

      if da.AnnotationList != nil {
        for _, annotation := range da.AnnotationList.Annotations {
          aEnd,err := strconv.Atoi(annotation.End)
          if err != nil {
            log.Println("Error: Unable to convert end annotation",annotation.End,"to int")
            continue
          }
          aStart,err := strconv.Atoi(annotation.Start)
          if err != nil {
            log.Println("Error: Unable to convert end annotation",annotation.Start,"to int")
            continue
          }
          if aEnd-aStart == 0 {
            // docno, ATTRIBUTE or TAG,id, name, start , length (ignored for attribute), value (optional int64 for TAGs, string for attribute) , parent,debyg
            fmt.Fprintf(annWriter,"%s\tATTRIBUTE\t%d\t%s\t%d\t%d\t\t0\t\n",da.Filename,tagid,annotation.Label,aStart,aEnd-aStart)
          } else {
            fmt.Fprintf(annWriter,"%s\tTAG\t%d\t%s\t%d\t%d\t\t0\t\n",da.Filename,tagid,annotation.Label,aStart,aEnd-aStart)
          }
          tagid++
        }
      }
      docid++
    }
    fmt.Fprintf(ixWriter,"</corpus>\n</parameters>")
  }()

  itemListSize[itemListId] = len(il.Items)

  for _, s := range il.Items {
    requests <- s
    k++
  }

  close(requests)

  for {
    select {
      case <-block:
       numWorkers--
       log.Println("Progress: Worker completed,",numWorkers, "remaining")
       if numWorkers == 0 {
         close(annotationsProcessor)
         <-doneWriting
         return
        }
    }
  }
}
