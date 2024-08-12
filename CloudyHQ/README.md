Very rough draft of the README.md

Loader will create the schema registry, message registry, and the endpoints registry. And it will use the xreg spec model from the github repo directly.

After the loader finish loading the schema, message, and endpoints, it will start the server and listen to the port 8080.

Then we send the PUT request to the server with the following payload:

For endpoints registry, we send to http://localhost:8080/reg-Endpoints?nested nested is the query parameter that will tell the server to udpate the whole json file.
The content can be found in the file: CloudyHQ/endpoints.json


For schema registry, we send to http://localhost:8080/reg-Schema?nested nested is the query parameter that will tell the server to udpate the whole json file.
The content can be found in the file: CloudyHQ/schema.json

For message registry, we send to http://localhost:8080/reg-Message?nested nested is the query parameter that will tell the server to udpate the whole json file.
But unfortunately, I think there are some issues with the validation of the message registry. 

The error message is:
```
Invalid extension(s) in "metadata.attributes": datacontenttype,dataschema,source,subject,type
```