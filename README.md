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
- PUT/POST operations
- DELETE operations
- see if we can prepend Path with / or append it with /
- Don't allow creation of Resource w/o version - for HTTP PUT
- save latest highest "versionId" in Resource so we can assign a new value
  if not provided by the client
- transactions
- add complex filters testcases
- create a schema (xreg model) checker so we can remove all checks during
  the set() calls and just check the entire entity all at once before we
  call set()
- support for IfValues
- test/support changing the model
  - test for invalid changes
- stop treating "labels" as special
- see if we can move "#resource??" into the attributes struct
- support overriding spec defined attributes - like "format"
- write down the model for the spec defined attributes
- support the create/modify by/on attributes
- add checks for valid obj/map key names in new validation funcs ****
- copy all of the types tests in http from Reg to groups, resources and vers
