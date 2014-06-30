Alveo-Indri-Wrapper
----------------

This code is the wrapper for indri in the Alveo project. However, if you don't know
what it is already, then you don't want to install it.

Installation instructions:

You will need:

* golang, available from http://golang.org/
* git and mercurial
* An Indri installation

Installation instructions:

     #install the dependencies
	 
     go get code.google.com/p/gorest
	 go get github.com/Alveo/alveo-golang-rest-client/alveoapi
	 
	 # install this code
	 go get github.com/Alveo/alveo-indri-wrapper
     go install github.com/Alveo/alveo-indri-wrapper

Next, you'll need to edit the config file to include the location of the indri binaries and the port you want the wrapper to run on.
	 
	 # create and edit the config file
	 cd $GOPATH/src/github.com/TimothyJones/hcsvlab-consumer
	 cp config.json.defaul config.json
	 vim config.json

That's it! You can run it with:

    alveo-indri-wrapper
	
In the directory where `config.json` is stored.
   
#Further documentation

Further documentation is available in the `doc` directory.
