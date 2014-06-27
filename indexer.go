package main

import (
  "log"
  "os"
  "bufio"
  "path"
  "fmt"
  "io/ioutil"
  "errors"
  "encoding/json"
  "strconv"
  "github.com/Alveo/alveo-golang-rest-client/alveoapi"
)


type documentAnnotations struct {
  Filename string
  AnnotationList* alveoapi.AnnotationList
}

func worker(api alveoapi.Api,requests chan string,done chan int, annotationsProcessor chan *documentAnnotations,itemListHelper *ItemListHelper) {
  for r := range requests {
    item, erro := api.GetItemFromUri(r)
    if erro != nil {
      log.Println("Error: Worker encountered",erro)
      continue
    }
    fileName := item.Metadata["alveo:handle"]

    block := make(chan int,2)
    go func(item alveoapi.Item) {
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

    go func(item alveoapi.Item) {
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

    itemListHelper.IncrementProgress()

    close(block)
  }

  done <- 1
}



func obtainAndIndex(numWorkers int, itemListId int,apiBase string, apiKey string) (err error){
  log.Println("Progress: Checking itemlists to see if",itemListId, "is in progress")
  log.Println("Progress: Indexing itemlist",itemListId,"with number of workers:",numWorkers)
  api := alveoapi.Api{apiBase,apiKey}
  ver,err := api.GetVersion()
  if err != nil {
    return
  }

  if ver.Api_version != "Sprint_23_demo" {
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
  itemListHelper := NewItemListHelper(itemListId,apiKey)

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

    tn := NewTagNameConverter()

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

          writeTag := func (annoName string) {
            if aEnd-aStart == 0 {
              // docno, ATTRIBUTE or TAG,id, name, start , length (ignored for attribute), value (optional int64 for TAGs, string for attribute) , parent,debyg
              fmt.Fprintf(annWriter,"%s\tATTRIBUTE\t%d\t%s\t%d\t%d\t\t0\t\n",da.Filename,tagid,annoName,aStart,aEnd-aStart)
            } else {
              fmt.Fprintf(annWriter,"%s\tTAG\t%d\t%s\t%d\t%d\t\t0\t\n",da.Filename,tagid,annoName,aStart,aEnd-aStart)
            }
            tagid++
          }

          if annotation.Type != "" {
            annoName,err := tn.Name(annotation.Type)
            if err != nil {
              log.Println("Error: Unable to find a name for this annotation:",annotation.Type)
            } else {
              writeTag(annoName)
            }
          }
          if annotation.Label != "" {
            annoName,err := tn.Name(annotation.Label)
            if err != nil {
              log.Println("Error: Unable to find a name for this annotation:",annotation.Label)
            } else {
              writeTag(annoName)
            }
          }
        }
      }
    }
    fmt.Fprintf(ixWriter,"</corpus>")
    for field := range tn.Used {
      fmt.Fprintf(ixWriter,"<field><name>%s</name></field>\n",field)
    }
    fmt.Fprintf(ixWriter,"</parameters>")

    // write tagnames to a file
    tagNames, err := tn.Dump()
    if err != nil {
      log.Println("Error: Unable to marshall the tagnames:",err)
    } else {
      err = ioutil.WriteFile(path.Join(itemListHelper.ConfigLocation(),"tagNames.json"),tagNames,0600)
      if err != nil {
        log.Println("Error: Unable to write tagnames to file:",err)
      }
    }
    // write itemlist to a file
    itemListJson, err := json.Marshal(il)
    if err != nil {
      log.Println("Error: Unable to marshall the itemlist:",err)
    } else {
      err = ioutil.WriteFile(path.Join(itemListHelper.ConfigLocation(),"itemlist.json"),itemListJson,0600)
      if err != nil {
        log.Println("Error: Unable to write itemlist to file:",err)
      }
    }

  }()

  itemListHelper.SetSize(len(il.Items))

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
