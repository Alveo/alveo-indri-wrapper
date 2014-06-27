package main

import (
  "encoding/json"
)

type ErrorResponse struct {
  Class string `json:"type"`
  Err string `json:"error"`
}

type IndexResponse struct {
  Class string `json:"type"`
  IndexStartedTime string `json:"index_started_time"`
}

type IndexProgressResponse struct {
  Class string `json:"type"`
  ItemsDownloaded int `json:"items_downloaded"`
  TotalToDownload int `json:"total_items"`
  IndexComplete bool `json:"index_complete"`
  IndexCreatedTime string `json:"index_created_time"`
}

type ItemListsResponse struct {
  Class string `json:"type"`
  ItemLists []*ItemListTuple
}

type ItemListTuple struct {
  Name string `json:"name"`
  Id string `json:"id"`
}

type AllQueryResult struct {
  Class string `json:"type"`
  IndexCreatedTime string `json:"index_created_time"`
  Matches []*MatchItem
}

type DocQueryResult struct {
  Class string `json:"type"`
  IndexCreatedTime string `json:"index_created_time"`
  Matches []*MatchDoc
}

type MatchItem struct {
  DocId string `json:"docid"`
  Url string `json:"url"`
  Location int64 `json:"location"`
  Match string `json:"match"`
}

type MatchDoc struct {
  DocId string `json:"docid"`
  Url string `json:"url"`
  Start int64 `json:"start"`
  End int64 `json:"end"`
}


func stringError(err error) (string) {
  var response = ErrorResponse{"error",err.Error()}
  result, errMars := json.Marshal(response)
  if errMars != nil {
    return "{type: \"error\",message: \"Cannot marshal json error\"}"
  }
  return string(result)
}

