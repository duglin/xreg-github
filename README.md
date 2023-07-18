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
- test filtering  across groups - OR and AND
- test filtering for (non)empty values - e.g. filter=id=  filter=id
- test for filtering case-insensitive string compares
- test for filtering with string being part of value
- test for exact match of numerics, bools
- ID -> uID vs sID
- pagination
- GET of resources blobs - all 3 variants(in DB, URL, proxy URL)
- PUT/POST operations
