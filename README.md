# xreg-github

Implementation of the xRegistry spec.

Still a long way to go.

To run it:
```
# You need to have Docker installed

# Start a mysql server. Give it about 30 seconds to fully start
$ make mysql

# Run the tests & xreg server (creates a new DB each time):
$ make

or to use existing DB (no tests):
$ make start
```

Try it:
```
# In a browser go to:
  http://localhost:8080?reg

# Or just:
$ curl http://localhost:8080
$ curl http://localhost:8080?inline

# To run a mysql client to see the DBs:
$ make mysql-client
```

TODOs:
- add tests for multiple registries at the same time
- test filtering for (non)empty values - e.g. filter=id=  filter=id
  - empty complex types too
- test for filtering case-insensitive string compares
- test for filtering with string being part of value
- test for exact match of numerics, bools
- pagination
- see if we can prepend Path with / or append it with /
- save latest highest "versionId" in Resource so we can assign a new value
  if not provided by the client
- transactions
- add complex filters testcases
- create a schema (xreg model) checker so we can remove all checks during
  the set() calls and just check the entire entity all at once before we
  call set()
- test/support changing the model
  - test for invalid changes
- see if we can move "#resource??" into the attributes struct
- support overriding spec defined attributes - like "format"
- write down the model for the spec defined attributes
- support the create/modify by/on attributes
- add checks for valid obj/map key names in new validation funcs ****
- copy all of the types tests in http from Reg to groups, resources and vers
- test to make sure an ID in the body == ID in URL for reg and group
- support multiple resources/versions in POSTs
- add support for hasdocument
- add support for resource vs resourcebase64 - return what was on the PUT
- add support for "default" in the model
- see if we can get rid of/clean-up the FindRegistry stuff
- need to decide on the best tx isolation level
- test to ensure we can do 2 tx at the same time
- convert internal server errors into "panic" so any "error" returned is a user
  error
- See if we can remove JustSet and always Save each time since we have TXs now

- test for required fields (singular/complex) at all level
  - esp during creation time
  - did group, continue with res, ver
- move ParseRequest to http_stuff.go
