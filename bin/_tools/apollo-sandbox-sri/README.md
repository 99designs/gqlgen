It has been so long that the details slipped had my mind.

See #2581 for the original history of this Apollo Sandbox playground feature.

> This is a [Subresource Integrity](https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity) check, so we can follow that the MDN documentation [Subresource Integrity - Web security | MDN](https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity) to get the hash value locally to specify the RSI if we need to.
> 
> Or take the JS URL and run it through https://www.srihash.org/ with sha256 selected?
> 
> Or maybe downloaded that script locally and did:
> ```
> cat FILENAME.js | openssl dgst -sha256 -binary | openssl base64 -A
> ```
> Or
> ```
> shasum -b -a 256 FILENAME.js | awk '{ print $1 }' | xxd -r -p | base64
> ```
> 

However, that was a pain to have to continually manually this, so in #2686 @gitxiongpan we figured out:

> The url https://embeddable-sandbox.cdn.apollographql.com/ will allow you to list the contents of the S3 bucket. I made a dumb script to figure out the latest one from the S3 bucket and calculate the [Subresource Integrity](https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity). This script is https://gist.github.com/StevenACoffman/2f15cd2e64f107d1a9a5f10f9748e1b0 and when I run it:
> 
> CDN_FILE=https://embeddable-sandbox.cdn.apollographql.com/7212121cad97028b007e974956dc951ce89d683c/embeddable-sandbox.umd.production.min.js
> curl -s $CDN_FILE | openssl dgst -sha256 -binary | openssl base64 -A; echo
> 
> ldbSJ7EovavF815TfCN50qKB9AMvzskb9xiG71bmg2I=
> So instead of setting it to "_latest" and having to forego the subresource integrity check, let's just update both to that and now we can try to remember to periodically run this dumb script and update it. Ok?

And then we all forgot about it and never did anything with it ever again! 😆 

Running it now, it gives me:
```
CDN_FILE=https://embeddable-sandbox.cdn.apollographql.com/02e2da0fccbe0240ef03d2396d6c98559bab5b06/embeddable-sandbox.umd.production.min.js
curl -s $CDN_FILE | openssl dgst -sha256 -binary | openssl base64 -A; echo
```

