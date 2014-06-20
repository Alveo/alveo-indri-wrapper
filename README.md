HCSVlab-consumer
----------------

This code is the wrapper for indri in the HCSVLab project. However, if you don't know
what it is already, then you don't want to install it.

Installation instructions:

You will need:

* golang, available from http://golang.org/
* git and mercurial
* An Indri installation

Installation instructions:

     #install the dependencies
	 
     go get code.google.com/p/gorest
	 go get github.com/TimothyJones/hcsvlabapi
	 
	 # install this code
	 go get github.com/TimothyJones/hcsvlab-consumer
     go install github.com/TimothyJones/hcsvlab-consumer

Next, you'll want to edit the config file to include your API key, the location of the HCSVLab API, and the location of the indri binaries.
	 
	 # create and edit the config file
	 cd $GOPATH/src/github.com/TimothyJones/hcsvlab-consumer
	 cp config.json.defaul config.json
	 vim config.json

That's it! You can run it with:

    hcsvlab-consumer
	
In the directory where `config.json` is stored.
   
#Further documentation

Further documentation is available in the `doc` directory.
