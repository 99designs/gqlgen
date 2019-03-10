### fileupload example

This server demonstrates how to handle file upload

to run this server
```bash
go run ./example/fileupload/server/server.go
```

and open http://localhost:8087 in your browser
  

### Single file

#### Operations

```js
{
  query: `
    mutation($file: Upload!) {
      singleUpload(file: $file) {
        id
      }
    }
  `,
  variables: {
    file: File // a.txt
  }
}
```

#### cURL request

```shell
curl localhost:8087/query \
  -F operations='{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { id } }", "variables": { "file": null } }' \
  -F map='{ "0": ["variables.file"] }' \
  -F 0=@./example/fileupload./testfiles/a.txt
```


```shell
curl localhost:8087/query \
  -F operations='{ "query": "mutation ($req: UploadFile!) { singleUploadWithPayload(req: $req) { id } }", "variables": { "req": {"file": null, "id": 1 } } }' \
  -F map='{ "0": ["variables.req.file"] }' \
  -F 0=@./example/fileupload/testfiles/a.txt
```

#### Request payload

```
POST /query HTTP/1.1
Host: localhost:8087
User-Agent: curl/7.60.0
Accept: */*
Content-Length: 525
Content-Type: multipart/form-data; boundary=--------------------
----c259ddf1cd194033
=> Send data, 525 bytes (0x20d)
--------------------------c259ddf1cd194033
Content-Disposition: form-data; name="operations"

{ "query": "mutation ($file: Upload!) { singleUpload(file: $file
) { id } }", "variables": { "file": null } }
--------------------------c259ddf1cd194033
Content-Disposition: form-data; name="map"

{ "0": ["variables.file"] }
--------------------------c259ddf1cd194033
Content-Disposition: form-data; name="0"; filename="a.txt"
Content-Type: text/plain

Alpha file content.
--------------------------c259ddf1cd194033--
```

### File list

#### Operations

```js
{
  query: `
    mutation($files: [Upload!]!) {
      multipleUpload(files: $files) {
        id
      }
    }
  `,
  variables: {
    files: [
      File, // b.txt
      File // c.txt
    ]
  }
}
```

#### cURL request

```
curl localhost:8087/query \
  -F operations='{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { id } }", "variables": { "files": [null, null] } }' \
  -F map='{ "0": ["variables.files.0"], "1": ["variables.files.1"] }' \
  -F 0=@./example/fileupload/testfiles/b.txt \
  -F 1=@./example/fileupload/testfiles/c.txt
```

```
curl localhost:8087/query \
  -F operations='{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }' \
  -F map='{ "0": ["variables.req.0.file"], "1": ["variables.req.1.file"] }' \
  -F 0=@./example/fileupload/testfiles/b.txt \
  -F 1=@./example/fileupload/testfiles/c.txt
```

#### Request payload

```
--------------------------ec62457de6331cad
Content-Disposition: form-data; name="operations"

{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { id } }", "variables": { "files": [null, null] } }
--------------------------ec62457de6331cad
Content-Disposition: form-data; name="map"

{ "0": ["variables.files.0"], "1": ["variables.files.1"] }
--------------------------ec62457de6331cad
Content-Disposition: form-data; name="0"; filename="b.txt"
Content-Type: text/plain

Bravo file content.

--------------------------ec62457de6331cad
Content-Disposition: form-data; name="1"; filename="c.txt"
Content-Type: text/plain

Charlie file content.

--------------------------ec62457de6331cad--
```

