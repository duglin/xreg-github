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

- pagination
- add checks for valid obj/map key names in new validation funcs ****
- support multiple resources/versions in POSTs
- support overriding spec defined attributes - like "format"
- support changing the model - test for invalid changes
- support the create/modify by/on attributes
- add support for resource vs resourcebase64 - return what was on the PUT
- support non-string "RESOURCE" values (e.g. json)
- add tests for immutable attributes
- add a tx.AddVersion/GetVersion so people don't need to do it themselves
