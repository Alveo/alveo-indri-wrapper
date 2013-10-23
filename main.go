package main

import (
  "fmt"
  "log"
  "bufio"
  "os"
  "path"
  "strconv"
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
//      for _,doc := range item.Documents {
        data, err := api.Get(item.Primary_text_url)
        if err != nil {
          log.Fatal(err)
        }
        log.Println("Saving",fileName, "(",len(data),"bytes)")
        fo, err := os.Create(path.Join("data",fileName))
        if err != nil {
          log.Fatal(err)
        }
        // close fo on exit and check for its returned error
        defer func() {
          if err := fo.Close(); err != nil {
              log.Fatal(err)
          }
          log.Println("Finished",fileName)
        }()
        w := bufio.NewWriter(fo)
        written, err := w.Write(data)
        if err != nil {
          log.Fatal(err)
        }
        log.Println(written, "bytes written to",fileName)
        w.Flush()
 //     }
      block <- 1
    }(item)

    go func(item Item) {
      annotations, err := api.GetAnnotations(item)
      if err != nil {
        log.Fatal(err)
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
  AnnotationList *AnnotationList
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
    log.Println("Starting to annotate")

    // Create annotations writer
    annFo, err := os.Create("annotation.offsets")
    if err != nil {
      log.Fatal(err)
    }
    defer func() {
      doneWriting <- 1
    }()
    defer func() {
      if err := annFo.Close(); err != nil {
          log.Fatal(err)
      }
      log.Println("Closing annFo")
    }()
    annWriter := bufio.NewWriter(annFo)

    // Create index properties writer
    ixFo, err := os.Create("index.properties")
    if err != nil {
      log.Fatal(err)
    }
    defer func() {
      if err := ixFo.Close(); err != nil {
          log.Fatal(err)
      }
      log.Println("Closing ixFo")
    }()
    ixWriter := bufio.NewWriter(ixFo)

    fmt.Fprintf(ixWriter,"<parameters>\n<index>repo</index>\n")
    fmt.Fprintf(ixWriter,"<corpus>\n")
    fmt.Fprintf(ixWriter,"  <class>txt</class>\n")
    fmt.Fprintf(ixWriter,"  <annotations>annotation.offsets</annotations>\n")

    for da := range annotationsProcessor {
      fmt.Fprintf(ixWriter,"  <path>data/%s</path>\n",da.Filename)
      log.Println("writing annotations for",da.Filename)

      for _, annotation := range da.AnnotationList.Annotations {
        if int(annotation.End-annotation.Start) == 0 {
          fmt.Fprintf(annWriter,"%d\tannotation\t%d\t%s\t%d\t%d\t\t0\t\n",docid,tagid,annotation.Label,int(annotation.Start),int(annotation.End-annotation.Start))
        } else {
          fmt.Fprintf(annWriter,"%d\tTAG\t%d\t%s\t%d\t%d\t\t0\t\n",docid,tagid,annotation.Label,int(annotation.Start),int(annotation.End-annotation.Start))
        }
        tagid++
      }
      docid++
    }
    fmt.Fprintf(ixWriter,"</corpus>\n</parameters>")
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
