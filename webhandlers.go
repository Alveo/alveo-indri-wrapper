package main

import (
  "bufio"
  "bytes"
  "code.google.com/p/gorest"
  "encoding/json"
  "path/filepath"
  "errors"
  "fmt"
  "io/ioutil"
  "log"
  "os"
  "net/http"
  "net/url"
  "os/exec"
  "path"
  "reflect"
  "strconv"
  "strings"
  "time"
  "github.com/Alveo/alveo-golang-rest-client/alveoapi"
)

func getApiKey(rq *http.Request) (string, error) {
  apiCookie,err  := rq.Cookie("vlab-key")
  if err != nil {
    return "", err
  }
  return apiCookie.Value, nil
}

func getApiLocation(rq *http.Request) (string, error) {
  apiCookie,err  := rq.Cookie("vlab-api")
  if err != nil {
    return "", err
  }
  return apiCookie.Value, nil
}

func urlMarshall(v interface{}) ([]byte, error) {
  return nil, nil
}

func getUrlForDocId(apiLocation string, docId string) string {
  return apiLocation + "/catalog/" + docId
}

func urlUnMarshall(data []byte, v interface{}) error {
  fmt.Println("recieved",string(data))
  parsed, err := url.ParseQuery(string(data))
  if err != nil {
    return err
  }

  mp, ok := v.(*map[string][]string);
  if !ok {
    return errors.New("Supplied interface was "+reflect.ValueOf(v).Type().String() + " instead of map[string][]string")
  }
  *mp  = parsed
  return nil
}

func NewUrlMarshaller() *gorest.Marshaller{
   return &gorest.Marshaller{urlMarshall,urlUnMarshall}
}

// Service Definition
type IndriService struct {
  gorest.RestService `root:"/" consumes:"application/x-www-form-urlencoded"`
  query  gorest.EndPoint `method:"GET" path:"/indri/query/doc/{itemList:int}/{query:string}" output:"string"`
  queryall  gorest.EndPoint `method:"GET" path:"/indri/query/all/{itemList:int}/{query:string}" output:"string"`
  index    gorest.EndPoint `method:"GET" path:"/indri/index/{itemList:int}" output:"string"`
  itemlists    gorest.EndPoint `method:"GET" path:"/indri/itemlists/" output:"string"`
  progress gorest.EndPoint `method:"GET" path:"/indri/progress/{itemList:int}/{after:string}" output:"string"`
  web gorest.EndPoint `method:"GET" path:"/indri/{url:string}" output:"string"`
  annotations gorest.EndPoint `method:"GET" path:"/indri/annotations/{itemList:int}" output:"string"`
  begin gorest.EndPoint `method:"POST" path:"/indri/" postdata:"map[string]"`
}

func(serv IndriService) Annotations(itemList int) string{
  apiKey, err := getApiKey(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API key specified"))
  }
  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")
  itemListHelper := NewItemListHelper(itemList,apiKey)

  annotationsJson, err := ioutil.ReadFile(path.Join(itemListHelper.ConfigLocation(),"tagNames.json"))
  if err != nil {
    return stringError(err)
  }

  return string(annotationsJson)
}


func(serv IndriService) Itemlists() string{
  apiKey, err := getApiKey(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API key specified"))
  }
  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")
  itemListHelper := NewItemListHelper(0,apiKey)

  baseDir := filepath.Dir(itemListHelper.ConfigLocation())

  files, _ := ioutil.ReadDir(baseDir)

  var res ItemListsResponse
  res.Class = "itemlists"
  res.ItemLists = make([]*ItemListTuple, 0, len(files))

  for _, f := range files {
    if f.IsDir() {
      itemListJson, err := ioutil.ReadFile(path.Join(baseDir,f.Name(),"itemlist.json"))
      var il alveoapi.ItemList
      if err == nil {
        err = json.Unmarshal(itemListJson,&il)
        if err != nil {
          log.Println("Error: Couldn't unmarshal json from",path.Join(baseDir,f.Name(),"itemlist.json"))
        } else {
          item := &ItemListTuple{il.Name,f.Name()}
          res.ItemLists = append(res.ItemLists,item)
        }
      }
    }
  }
  result, errMars := json.Marshal(res);
  if errMars != nil {
    return stringError(errMars)
  }
  return string(result)
}



func(serv IndriService) Query(itemList int, query string) string{
  log.Println("Info: Query for doc matches received:",query)
  apiKey, err := getApiKey(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API key specified"))
  }
  apiLoc, err := getApiLocation(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API location specified"))
  }
  itemListHelper := NewItemListHelper(itemList,apiKey)
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
      log.Println("Error: response contains less than four fields (",scanner.Text(),")")
      if strings.Contains(scanner.Text(),"Couldn't understand this query") {
        log.Println("Error: Indri did not understand the query")
        return stringError(errors.New("Indri did not understand the query"))
      }
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
      match := &MatchDoc{docId,getUrlForDocId(apiLoc,docId),start,end}
      res.Matches = append(res.Matches,match)
    }
  }
  result, errMars := json.Marshal(res);
  if errMars != nil {
    return stringError(errMars)
  }
  str := string(result)
  return str
}

func(serv IndriService) Queryall(itemList int, query string) string{
  log.Println("Info: Query all recieved request for itemlist",itemList, " with query",query)
  apiKey, err := getApiKey(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API key specified"))
  }
  apiLoc, err := getApiLocation(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API location specified"))
  }
  itemListHelper := NewItemListHelper(itemList,apiKey)
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
      item := &MatchItem{docId,getUrlForDocId(apiLoc,docId),location,match}
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


func(serv IndriService) Index(itemList int) string{
  log.Println("Info: Request to index itemList",itemList)
  apiKey, err := getApiKey(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API key specified"))
  }
  apiLoc, err := getApiLocation(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API location specified"))
  }

  itemListHelper := NewItemListHelper(itemList,apiKey)
  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")
  // Declare upfront because of use of goto
  cmd := exec.Command(config.Binaries.IndriBuildIndex, path.Join(itemListHelper.ConfigLocation(),"index.properties"))
  var out bytes.Buffer


  err = itemListHelper.BeginIndexingProgress()
  if err != nil {
    return stringError(err)
  }

  go func() {
    defer func() {
 //     progressMutex.Lock()
//      itemListsInProgress[itemList] = 0
//      progressMutex.Unlock()
    }()

    // processing begins here
    log.Println("Info: API Key is ", itemListHelper.Key)
    err = obtainAndIndex(10,itemList,apiLoc,itemListHelper.Key)
    if err != nil {
      goto errHandle
    }

    log.Println("Progress: Removing old index")
    err = itemListHelper.RemoveRepo()
    if err != nil {
      goto errHandle
    }

    err = itemListHelper.MkdirRepo()
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

    itemListHelper.SetIndexingError(err)
    return
  }()

  res := &IndexResponse{"indexing",time.Now().Format(TimeFormat)}

  result, errMars := json.Marshal(res);
  if errMars != nil {
    return stringError(errMars)
  }
  return string(result)
}

func(serv IndriService) Progress(itemList int,after string) string{
  log.Println("Info: Index progress requested for itemlist",itemList)

  apiKey, err := getApiKey(serv.Context.Request())
  if err != nil {
    return stringError(errors.New("No API key specified"))
  }
  itemListHelper := NewItemListHelper(itemList,apiKey)

  serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
  serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")

  numProcessed, inProgress, err := itemListHelper.GetProgress()

  if ! inProgress {
    return stringError(errors.New("Indexing not in progress"))
  }

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

  res := IndexProgressResponse{"progress",numProcessed,itemListHelper.GetSize(),completed,createdTime}

  result, errMars := json.Marshal(res);
  if errMars != nil {
    return stringError(errMars)
  }
  return string(result)
}


func(serv IndriService) Web(url string) string {
  log.Println("Info: Asked to serve",url)
  url = strings.TrimLeft(url,"/\\.")
  begin, err := ioutil.ReadFile(path.Join(config.WebDir,path.Clean(url)))
  if err != nil {
    if os.IsNotExist(err) {
      serv.ResponseBuilder().SetResponseCode(404)
    }
    serv.ResponseBuilder().SetHeader("Access-Control-Allow-Origin","*")
    serv.ResponseBuilder().SetContentType("application/json; charset=\"utf-8\"")
    return stringError(err)
  }

  if strings.HasSuffix(url,".js") {
    serv.ResponseBuilder().SetContentType("text/javascript; charset=\"utf-8\"")
  } else if strings.HasSuffix(url,".css") {
    serv.ResponseBuilder().SetContentType("text/css; charset=\"utf-8\"")
  } else if strings.HasSuffix(url,".png") {
    serv.ResponseBuilder().SetContentType("image/png;")
  } else {
    serv.ResponseBuilder().SetContentType("text/html; charset=\"utf-8\"")
  }

  return string(begin)
}


func(serv IndriService) Begin(PostData map[string][]string) {
  log.Println("Info: Asked to kickoff: ",PostData)
  key, ok := PostData["api_key"]
  if ! ok {
    serv.ResponseBuilder().SetResponseCode(400)
    return
  }
  apiLocation, ok := PostData["item_list_url"]
  if ! ok {
    serv.ResponseBuilder().SetResponseCode(400)
    return
  }

  if len(apiLocation) == 0 || len(key) == 0 {
    serv.ResponseBuilder().SetResponseCode(400).WriteAndOveride([]byte("Missing API or key"))
    return
  }

  idxSlash := strings.LastIndex(apiLocation[0],"/")
  if idxSlash == -1 {
    serv.ResponseBuilder().SetResponseCode(400).WriteAndOveride([]byte("No slash for the itemList number"))
    return
  }
  itemListIdString := apiLocation[0][idxSlash+1:]
  apiBase := strings.TrimSuffix(apiLocation[0],"/item_lists/" + itemListIdString)
  log.Println("Info: Setting apiBase to",apiBase, "from", apiLocation[0])
  itemListIdString = strings.TrimSuffix(itemListIdString,".json")

  serv.ResponseBuilder().AddHeader("Set-Cookie","vlab-action-itemlist=" + itemListIdString)
  serv.ResponseBuilder().AddHeader("Set-Cookie","vlab-api=" + apiBase)
  serv.ResponseBuilder().AddHeader("Set-Cookie","vlab-key=" + key[0])
  serv.ResponseBuilder().SetResponseCode(301).Location("/indri/begin.html")
  return
}


