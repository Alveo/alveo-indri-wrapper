package main

import (
  "path"
  "strconv"
  "os"
)

type ItemListHelper struct {
  Id int
}

// Returns the location for the index
func (il *ItemListHelper) RepoLocation() string {
  return path.Join("repos",strconv.FormatInt(int64(il.Id),10))
}

// Returns the location for the collection to index
func (il *ItemListHelper) DataLocation() string {
  return path.Join("data",strconv.FormatInt(int64(il.Id),10))
}

// Deletes the index directory for this collection 
func (il *ItemListHelper) RemoveRepo() (err error) {
  err = os.RemoveAll(il.RepoLocation())
  return
}

// Deletes the data directory for this collection
func (il *ItemListHelper) RemoveData() (err error) {
  err = os.RemoveAll(il.DataLocation())
  return
}

// Creates the data directory for this collection
func (il *ItemListHelper) MkdirData() (err error) {
  err = os.MkdirAll(il.DataLocation(),os.ModeDir)
  return
}



