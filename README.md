# xreg-github

Implementation of the xRegistry spec.

Still a long way to go.

To run it:
```
# You need to have Docker installed

# Build, test and run the xreg server (creates a new DB each time):
$ make

or to use existing DB (no tests):
$ make start
```

Try it:
```
# In a browser go to:
  http://localhost:8080?ui

# Or just:
$ curl http://localhost:8080
$ curl http://localhost:8080?inline

# To run a mysql client to see the DBs (debugging):
$ make mysql-client
```

# Developers

See `misc/Dockefile-dev` for the minimal things you'll need to install.
Useful Makefile targets:
```
- make              : build, test and run the server locally
- make all          : build, test and run the server locally
- make run          : build the exes and run it (no tests)
- make test         : build the exes and run tests, don't run
- make clean        : erase all build artifacts, stop mysql. Basically, reset
- make image        : build the all Docker images
- make push         : push the Docker images to DockerHub
- make mysql        : just start mysql as a Docker container
- make mysql-client : run the mysql client, for testing
- make testdev      : build a dev docker image, and build/test/run everything
                      to make sure the minimal dev install requirements
                      haven't changed
```

OLD TODO:
- Move the logic that takes the Path into account for the query into
  GenerateQuery
- Make sure that the Path entity is always in the result set when filtering
- twiddle the self and XXXUrls to include proper filter and inline stuff
- see if we can get rid of the recursion stuff
- should we add "/" to then end of the Path for non-collections, then
  we can just look for PATH/%  and not PATH + PATH/%
- can we set the registry's path to "" instead of NULL ?? already did, test it
- add support for boolean types (set/get/filter)

TODOs:
- add tests for multiple registries at the same time
- test filtering for (non)empty values - e.g. filter=id=  filter=id
  - empty complex types too
- test for filtering case-insensitive string compares
- test for filtering with string being part of value
- test for exact match of numerics, bools
- add complex filters testcases
- copy all of the types tests in http from Reg to groups, resources and vers
- test to make sure an ID in the body == ID in URL for reg and group
- test to ensure we can do 2 tx at the same time
- need to decide on the best tx isolation level
- add http test for maxVersions (already have non-http tests)

- see if we can prepend Path with / or append it with /
- see if we can remove all uses of JustSet except for the few testing cases
  where we need to set a property w/o verifying/saving it
  - Just see if we can clean-up the Set... stuff in general
- convert internal errors into "panic" so any "error" returned is a user error
- see if we can move "#resource??" into the attributes struct
- fix it so that a failed call to model.Save() (e.g. verify fails) doesn't
  invalidate existing local variables. See if we can avoid redownloading the
  model from the DB
- make sure we check for uniqueness of IDs - they're case insensitive and
  unique within the scope of their parent
- make sure we throw an error if ?specversion on HTTP requests specifies the
  wrong version

- pagination
- have DB generate the COLLECTIONcount attributes so people can query over
  them and we don't need the code to calculate them (can we due to filters?)
- add checks for valid obj/map key names in new validation funcs ****
- support overriding spec defined attributes - like "format"
- support changing the model - test for invalid changes
- add tests for immutable attributes
- support the resource sticky/default attributes
  - remove ?setdefault.. for some apis
  - process ?setdefaultversionid flag before we update things
  - make sure we support them as http headers too
- support createdat and modifiedat as http headers
- make sure we don't let go http add "content-type" header for docs w/o a value
- add support for PUT / to update the model
- add model tests for typemap - just that we can set via full model updates
- support the DB vanishing for a while
- create an UpdateDefaultVersion func in resource.go to move it from http logic
- support ?export
- test xref more
- support readonly - remove resource.readonly
- remove "readonly" attribute from model Resources, add to Resource
- format is optional in schema spec
- s/format/envelope/
- s/message/protocoloptions/
- s/binding/protocol/
- add envelopeoptions and envelopmetadata
- allow $meta on hasdoc=false resources
- add "validation" attribute - schema
- error w/o ?nested + children
- Add content-disposition header for hasdoc resources
- s/?resource/?attrib support
- move epoch to be resource level
- make sure ID only has valid chars
