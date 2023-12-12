# xreg-github

Implementation of the xRegistry spec.

Still a long way to go.

To run it:
```
# You need to have Docker installed

# Start a mysql server. Give it about 30 seconds to fully start
$ make mysql

# Run the xreg server (creates a new DB each time):
$ make

or to use existing DB:
$ make start

# Try it:
$ curl http://localhost:8080
$ curl http://localhost:8080?inline

# To run a mysql client:
$ make mysql-client
```

TODOs:
- add tests for multiple registries at the same time
- test filtering for (non)empty values - e.g. filter=id=  filter=id
- test for filtering case-insensitive string compares
- test for filtering with string being part of value
- test for exact match of numerics, bools
- pagination
- GET of resources blobs - all 3 variants(in DB, URL, proxy URL)
- PUT/POST operations
- see if we can prepend Path with / or append it with /
- Don't allow creation of Resource w/o version - for HTTP PUT
- save latest highest "versionId" in Resource so we can assign a new value
  if not provided by the client
- transactions
- add complex filters testcases
