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
  -F operations='{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { id, name, content } }", "variables": { "file": null } }' \
  -F map='{ "0": ["variables.file"] }' \
  -F 0=@./example/fileupload./testfiles/a.txt
```

#### Request payload

```
POST /query HTTP/1.1
Host: localhost:8087
User-Agent: curl/7.60.0
Accept: */*
Content-Length: 540
Content-Type: multipart/form-data; boundary=--------------------
----e6b2b29561e71173

=> Send data, 540 bytes (0x21c)
--------------------------e6b2b29561e71173
Content-Disposition: form-data; name="operations"

{ "query": "mutation ($file: Upload!) { singleUpload(file: $file
) { id, name, content } }", "variables": { "file": null } }
--------------------------e6b2b29561e71173
Content-Disposition: form-data; name="map"

{ "0": ["variables.file"] }
--------------------------e6b2b29561e71173
Content-Disposition: form-data; name="0"; filename="a.txt"
Content-Type: text/plain

Alpha file content.
--------------------------e6b2b29561e71173--
```

### Single file with payload

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
  -F operations='{ "query": "mutation ($req: UploadFile!) { singleUploadWithPayload(req: $req) { id, name, content } }", "variables": { "req": {"file": null, "id": 1 } } }' \
  -F map='{ "0": ["variables.req.file"] }' \
  -F 0=@./example/fileupload/testfiles/a.txt
```

#### Request payload

```
POST /query HTTP/1.1
Host: localhost:8087
User-Agent: curl/7.60.0
Accept: */*
Content-Length: 575
Content-Type: multipart/form-data; boundary=--------------------
----38752760889d14aa

=> Send data, 575 bytes (0x23f)
--------------------------38752760889d14aa
Content-Disposition: form-data; name="operations"

{ "query": "mutation ($req: UploadFile!) { singleUploadWithPaylo
ad(req: $req) { id, name, content } }", "variables": { "req": {"
file": null, "id": 1 } } }
--------------------------38752760889d14aa
Content-Disposition: form-data; name="map"

{ "0": ["variables.req.file"] }
--------------------------38752760889d14aa
Content-Disposition: form-data; name="0"; filename="a.txt"
Content-Type: text/plain

Alpha file content.
--------------------------38752760889d14aa--
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
  -F operations='{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { id, name, content } }", "variables": { "files": [null, null] } }' \
  -F map='{ "0": ["variables.files.0"], "1": ["variables.files.1"] }' \
  -F 0=@./example/fileupload/testfiles/b.txt \
  -F 1=@./example/fileupload/testfiles/c.txt
```

#### Request payload

```
POST /query HTTP/1.1
Host: localhost:8087
User-Agent: curl/7.60.0
Accept: */*
Content-Length: 742
Content-Type: multipart/form-data; boundary=--------------------
----d7aca2a93c3655e0

=> Send data, 742 bytes (0x2e6)
--------------------------d7aca2a93c3655e0
Content-Disposition: form-data; name="operations"

{ "query": "mutation($files: [Upload!]!) { multipleUpload(files:
 $files) { id, name, content } }", "variables": { "files": [null
, null] } }
--------------------------d7aca2a93c3655e0
Content-Disposition: form-data; name="map"

{ "0": ["variables.files.0"], "1": ["variables.files.1"] }
--------------------------d7aca2a93c3655e0
Content-Disposition: form-data; name="0"; filename="b.txt"
Content-Type: text/plain

Bravo file content.
--------------------------d7aca2a93c3655e0
Content-Disposition: form-data; name="1"; filename="c.txt"
Content-Type: text/plain

Charlie file content.
--------------------------d7aca2a93c3655e0--
```



### File list with payload

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
  -F operations='{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id, name, content } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }' \
  -F map='{ "0": ["variables.req.0.file"], "1": ["variables.req.1.file"] }' \
  -F 0=@./example/fileupload/testfiles/b.txt \
  -F 1=@./example/fileupload/testfiles/c.txt
```

#### Request payload

```
=> Send header, 191 bytes (0xbf)
POST /query HTTP/1.1
Host: localhost:8087
User-Agent: curl/7.60.0
Accept: */*
Content-Length: 799
Content-Type: multipart/form-data; boundary=--------------------
----65aab09fb49ee66f

=> Send data, 799 bytes (0x31f)
--------------------------65aab09fb49ee66f
Content-Disposition: form-data; name="operations"

{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithP
ayload(req: $req) { id, name, content } }", "variables": { "req"
: [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }
--------------------------65aab09fb49ee66f
Content-Disposition: form-data; name="map"

{ "0": ["variables.req.0.file"], "1": ["variables.req.1.file"] }
--------------------------65aab09fb49ee66f
Content-Disposition: form-data; name="0"; filename="b.txt"
Content-Type: text/plain

Bravo file content.
--------------------------65aab09fb49ee66f
Content-Disposition: form-data; name="1"; filename="c.txt"
Content-Type: text/plain

Charlie file content.
--------------------------65aab09fb49ee66f--
```

