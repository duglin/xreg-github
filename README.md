# xreg-github

Implementation of the xRegistry spec.

Still a long way to go.

To run it:
```
# You need to have Docker installed

# Start a mysql server. Give it about 30 seconds to fully start
$ make mysql

# Run the xreg server
$ make

# Try it:
$ curl http://localhost:8080
$ curl http://localhost:8080?inline
```
