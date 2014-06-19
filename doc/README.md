Alveo Indri Wrapper User Documentation
--------------------------------------

#Setup

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

#Structure of the wrapper

The wrapper provides a RESTful json API to handle Alveo itemlists. It has hooks to create Indri indexes from Alveo Itemlists, and two different ways of querying those indexes.
Offset metadata is handled by default, and is available at query time.

#API

The Alveo API location and key are both provided to the wrapper using cookies. The cookies are named `vlab-key` and `vlab-api` respectively.

 indri/index/{itemList:int}

 indri/query/doc/{itemList:int}/{query:string}
 indri/query/all/{itemList:int}/{query:string}
 indri/progress/{itemList:int}/{after:string}
 indri/{url:string}

##Kickoff with a POST request

`item_list_url` and `api_key`

#Web

todo web page overview


todo describe the lifecycle of an indri index in this system
