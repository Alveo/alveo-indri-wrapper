package main

import (
  "strings"
  "unicode"
  "fmt"
  "errors"
  "log"
)

type TagNameConverter struct {
  Names map[string]string
  Used map[string]struct{}
}

func NewTagNameConverter() *TagNameConverter {
  names := make(map[string]string)
  used := make(map[string]struct{})
  return &TagNameConverter{names,used}
}

func isNotLower(r rune) rune {
  if unicode.IsLower(r) {
    return r
  }
  return -1
}

func (tn *TagNameConverter) Name(original string) (string, error) {
  if existingName, ok := tn.Names[original] ; ok {
    //log.Println("Info: Annotation",original,"is already",existingName)
    return existingName, nil
  }

  name := original

  if slashPos := strings.LastIndex(name, "/") ; slashPos != -1 {
    name = name[slashPos:]
  }
  name = strings.ToLower(name)
  name = strings.Map(isNotLower,name)

  queue := make(chan string,1000)
  queue <- name

  for q := range queue {
    //log.Printf("Info: Testing possible annotation name '%s'\n",q)
    if _, isSet := tn.Used[q] ; ! isSet {
      log.Println("Info: Annotation",original,"becomes",q)
      tn.Names[original] = q
      tn.Used[q] = struct{}{}
      return q, nil
    }
    for r := 'a' ; r <= 'z' ; r += 1 {
      select {
        case queue <- fmt.Sprintf("%s%c",q,r):
        default:
          // it's ok to drop some potential names, so if the channel is full, we just skip this name
      }
    }
  }
  log.Println("Error: Failed to find a name for Annotation",original)
  return "", errors.New("Failed to find a name for Annotation " + original)
}
