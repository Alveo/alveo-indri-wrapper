Alveo Indri Wrapper User Documentation
--------------------------------------

Throughout this document, the following terms are used:

* Index or indri index: this refers to the search index built by the wrapper. It is stored internally by the wrapper, and can be queried by users of this wrapper.

* Itemlist: this refers to a list of documents created inside Alveo. It is what is passed to the wrapper to create indexes.

## Setup

For installation, follow the instructions in readme.md in the root folder of this project.

Here is an annotated version of `config.json`

     {
       // The Binaries object contains the location of each binary used by this project
       // The Indri binaries are available in an indri installation,
       // while QueryAll is specific to this project. Installation instructions are in readme.md
       "Binaries": {                   
         "IndriBuildIndex":"/path/to/IndriBuildIndex",
         "QueryAll":"/path/to/QueryAll",
         "IndriRunQuery":"/path/to/IndriRunQuery"
       },
       // This is the directory where the wrapper should find the web pages to serve.
       // You probably don't need to change it.
       "WebDir":"web"
       // This is the port that the wrapper should serve content on. 
       "Port":"8787"
     }

## Structure of the wrapper

The wrapper provides a RESTful json API to handle Alveo itemlists. It has hooks to create Indri indexes from Alveo Itemlists, and two different ways of querying those indexes.
Offset metadata is handled by default, and is available at query time.

## Index uniqueness

Indexes are stored using a tuple of the API key and itemlist ID as the key. This means that each index of an itemlist 
is unique for a particular Alveo key. If other Alveo users with different API keys want to query an indri index, they
will have to ask the wrapper to index another copy.

Note that there is no versioning of itemlists- itemlists are not checked to see if they have changed when an index is
queried. However, all query responses include the time that the index was created. This can be used by users to
determine whether a re-index is required.

## Reindexing itemlists

If you wish to manually rebuild an itemlist index from the copy currently stored inside Alveo,
you can do this from the `index a new itemlist` tab. Alternatively, you can use the REST API, by sending a request to `/indri/index/{itemlist id}`.

In the web interface, you will be taken to a progress page, which will update with the current
download and indexing progress (or any fatal errors encountered during processing and indexing).
If using the REST API, you will instead be given a timestamp which you can use to determine the 
current indexing progress, using the progress URL.

If two users with the same API key try to index the same itemlist at the same time,
the second user will receive an error.

While the itemlist content, documents and metadata are downloading, the index will still be
available for querying. However, once indexing begins, the old index is no longer available.
During this time, queries on the old index will fail due to a missing index.

##API

The wrapper also provides a JSON API. The web interface is built on top of the JSON API, so anything that can be done with the web interface can also be done with the Wrapper's API.

The Alveo API location and key are both provided to the wrapper using cookies. The cookies are named `vlab-key` and `vlab-api` respectively. All requests require these cookies to be set by the client.

All responses from the API include a `type` field. This can be used to determine whether the response is a content response or an error response.

* `indri/index/{itemList:int}`: This URL begins the indexing of a particular itemlist. Here is an example response:

     {
      "type":"indexing", // Always 'indexing' when there are no errors
      "index_started_time":"Fri, 20 Jun 2014 12:03:54 EST" // The timestamp that indexing was started at
     }

* `indri/progress/{itemList:int}/{timestamp:string}`: This URL is used to determine the indexing progress. Use the timestamp provided from a call to `index`. For example: 
  `indri/progress/11/Fri, 20 Jun 2014 12:03:54 EST`. A typical response is:

     {
       "type":"progress", // Always 'progress' when there are no errors
       "items_downloaded":50, // How many items have been downloaded from Alveo
       "total_items":50, // How many items there are in this itemlist
       "index_complete":true, // true if indexing is complete, false otherwise.
       "index_created_time":"Fri, 20 Jun 2014 12:04:46 EST" // if the index is complete, this is the time it was completed at
     }

* `indri/query/doc/{itemList:int}/{query:string}`: Used to query an index for document matches. A typical reponse is:

      {
         "type":"result-doc", // Always 'result-doc' when there are no errors
         "index_created_time":"Fri, 20 Jun 2014 12:04:46 EST", // The time that the index was created
         "Matches":[ // An array containing all the document matches
           {
             "docid":"collection:docid", // The collection and docid pair used within Alveo
             "url":"https://path.to.document/in/alveo", // The path to the document in Alveo.
             "start":0,     // The start and the end of the match
             "end":1610     // Since this is a document match, this is the start 
                            // and the end of the document. Measured in bytes.
           },
          .... // Further matches
        ]
     }

* `indri/query/all/{itemList:int}/{query:string}`: Used to query a collection for all matches. This returns one match per document. A typical response is:

      {
        "type":"result-all", // Always 'result-all' when there are no errors
        "index_created_time":"Fri, 20 Jun 2014 12:04:46 EST", // The time that the index was created
        "Matches":[ // an array containing all the document matches
          {
            "docid":"collection:docid", // The collection and docid pair used within Alveo
            "url":"https://path.to.document/in/alveo", // The path to the document in Alveo.
            "location":1271, // The location in the document where the match starts (in bytes)
            "match":" .... context for the match ..."
          }
        ]
        /* Further matches go here */
      }


###Kickoff with a POST request

If you wish to write a program to send a user to the web interface, you can also send a POST request to `/indri/` with the following variables set:

* `item_list_url`: Set to the URL for the itemlist in Alveo
* `api_key`: Set to the user's Alveo API key.

An example of this form is avaiable in the HTML of `/indri/kickoff.html`

##Web interface

There's also a web interface to the above functions, allowing users to interact with indri indexes of itemlists in the browser. The key locations are:

* `/indri/index.html`: The main page allowing the user to submit queries
* `/indri/kickoff.html`: A page allowing the user to begin the indexing of a particular itemlist.
