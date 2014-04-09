package main

import (
  "path"
  "sync"
  "errors"
  "strings"
  "time"
  "strconv"
  "os"
  "log"
)

type ItemListHelper struct {
  Id int
  Key string
}

var itemListLocks struct {
  itemListsInProgress map[int]int
  errorsFromIndex map[int]error // there can only be one error per index request, so that's ok.
  itemListSize map[int]int
  progressMutex sync.Mutex
}

const TimeFormat = time.RFC1123

func initialiseLocks() {
  itemListLocks.itemListsInProgress = make(map[int]int)
  itemListLocks.itemListSize = make(map[int]int)
  itemListLocks.errorsFromIndex = make(map[int]error)
}

// Increments the indexing progress counter for this itemlist
func (il *ItemListHelper) IncrementProgress() {
  itemListLocks.progressMutex.Lock()
  itemListLocks.itemListsInProgress[il.Id]++
  itemListLocks.progressMutex.Unlock()
}

// Returns the number indexed, whether the indexing is currently progressing, and whether 
// there have been any errors for the indexing of this itemlist
func (il *ItemListHelper) GetProgress() (indexed int,inProgress bool, err error) {
  itemListLocks.progressMutex.Lock()
  indexed, inProgress = itemListLocks.itemListsInProgress[il.Id]
  err = itemListLocks.errorsFromIndex[il.Id]
  itemListLocks.progressMutex.Unlock()
  return
}

func (il *ItemListHelper) SetSize(size int) {
  itemListLocks.progressMutex.Lock()
  itemListLocks.itemListSize[il.Id] = size
  itemListLocks.progressMutex.Unlock()
}

func (il *ItemListHelper) GetSize() (int) {
  itemListLocks.progressMutex.Lock()
  ret := itemListLocks.itemListSize[il.Id]
  itemListLocks.progressMutex.Unlock()
  return ret
}


func (il *ItemListHelper) SetIndexingError(err error) {
  itemListLocks.progressMutex.Lock()
  itemListLocks.errorsFromIndex[il.Id] = err
  itemListLocks.progressMutex.Unlock()
}


// Set the state for beginning the indexing progress for this itemlist. 
// If indexing is currently in progress for this itemlist, this function returns an error
// Clears any previous indexing errors if successful.
func (il *ItemListHelper) BeginIndexingProgress() (error){
  itemListLocks.progressMutex.Lock()

  if itemListLocks.itemListsInProgress[il.Id] != 0 {
    log.Println("Error: Indexing already in progress")
    err := errors.New("Itemlist is already being indexed. Please wait for the indexing to complete")
    itemListLocks.progressMutex.Unlock()
    return err
  }
  itemListLocks.itemListsInProgress[il.Id] = 1
  delete(itemListLocks.errorsFromIndex,il.Id)

  itemListLocks.progressMutex.Unlock()
  return nil
}


// Returns the location for the index
func (il *ItemListHelper) RepoLocation() string {
  return path.Join("repos",il.Key,strconv.FormatInt(int64(il.Id),10))
}

// Returns the location for the collection to index
func (il *ItemListHelper) DataLocation() string {
  return path.Join("data",il.Key,strconv.FormatInt(int64(il.Id),10))
}

// Returns the location for the config files
func (il *ItemListHelper) ConfigLocation() string {
  return path.Join("config",il.Key,strconv.FormatInt(int64(il.Id),10))
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
  err = os.MkdirAll(il.DataLocation(),os.ModeDir | 0755)
  return
}

// Deletes the config directory for this collection
func (il *ItemListHelper) RemoveConfig() (err error) {
  err = os.RemoveAll(il.ConfigLocation())
  return
}

// Creates the config directory for this collection
func (il *ItemListHelper) MkdirConfig() (err error) {
  err = os.MkdirAll(il.ConfigLocation(),os.ModeDir | 0755)
  return
}

// Creates the repo directory for this collection
func (il *ItemListHelper) MkdirRepo() (err error) {
  err = os.MkdirAll(il.RepoLocation(),os.ModeDir | 0755)
  return
}

// Returns the time that the index was created. Returns an error if the
// index does not exist
func (il *ItemListHelper) CreatedTime() (time string,err error) {
  fi, err := os.Lstat(path.Join(il.RepoLocation(),"manifest"))
  if err != nil {
    err = errors.New("No index found for this itemlist")
    return
  }

  time = fi.ModTime().Format(TimeFormat)
  return
}

// Returns the document handle that is represented by this path.
// Used for converting between the document in the results and
// the document handle understood by the vlab server
func (il *ItemListHelper) docIdForFile(filename string) string {
  return strings.TrimPrefix(filename,path.Join(il.DataLocation()))[1:]
}


// Convienience method for preparing the system for download.
// Removes the old data and config directories (if any), and then
// recreates them.
func (il *ItemListHelper) MakeReadyForDownload() (err error) {
  err = il.RemoveData()
  if err != nil {
    return
  }

  err = il.RemoveConfig()
  if err != nil {
    return
  }

  err = il.MkdirData()
  if err != nil {
    return
  }

  err = il.MkdirConfig()
  return
}

