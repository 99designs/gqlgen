package playground

import (
	"encoding/json"
	"html/template"
	"net/http"
)

var altairPage = template.Must(template.New("altair").Parse(`<!doctype html>
<html>

<head>
  <meta charset="utf-8">
  <title>{{.title}}</title>
  <base href="https://cdn.jsdelivr.net/npm/altair-static@{{.version}}/build/dist/">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <link rel="icon" type="image/x-icon" href="favicon.ico">
  <link href="styles.css" rel="stylesheet" crossorigin="anonymous" integrity="{{.cssSRI}}"/>
</head>

<body>
  <app-root>
    <style>
      .loading-screen {
        display: none;
      }
    </style>
    <div class="loading-screen styled">
      <div class="loading-screen-inner">
        <div class="loading-screen-logo-container">
          <img src="assets/img/logo_350.svg" alt="Altair">
        </div>
        <div class="loading-screen-loading-indicator">
          <span class="loading-indicator-dot"></span>
          <span class="loading-indicator-dot"></span>
          <span class="loading-indicator-dot"></span>
        </div>
      </div>
    </div>
  </app-root>

  <script rel="preload" as="script" type="text/javascript" crossorigin="anonymous" integrity="{{.mainSRI}}" src="main.js"></script>
  <script rel="preload" as="script" type="text/javascript" crossorigin="anonymous" integrity="{{.polyfillsSRI}}" src="polyfills.js"></script>
  <script rel="preload" as="script" type="text/javascript" crossorigin="anonymous" integrity="{{.runtimeSRI}}" src="runtime.js"></script>

  <script>
{{- if .endpointIsAbsolute}}
	const url = {{.endpoint}};
	const subscriptionUrl = {{.subscriptionEndpoint}};
{{- else}}
	const url = location.protocol + '//' + location.host + {{.endpoint}};
	const wsProto = location.protocol == 'https:' ? 'wss:' : 'ws:';
	const subscriptionUrl = wsProto + '//' + location.host + {{.endpoint}};
{{- end}}
    var altairOptions = {
        endpointURL: url,
        subscriptionsEndpoint: subscriptionUrl,
    };
	var options = {...altairOptions, ...JSON.parse({{.options}})};
    window.addEventListener("load", function() {
      AltairGraphQL.init(options);
    });
  </script>
</body>

</html>`))

// AltairHandler responsible for setting up the altair playground
func AltairHandler(title, endpoint string, options map[string]any) http.HandlerFunc {
	jsonOptions, err := json.Marshal(options)
	if err != nil {
		jsonOptions = []byte("{}")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := altairPage.Execute(w, map[string]any{
			"title":                title,
			"endpoint":             endpoint,
			"endpointIsAbsolute":   endpointHasScheme(endpoint),
			"subscriptionEndpoint": getSubscriptionEndpoint(endpoint),
			"version":              "8.1.3",
			"cssSRI":               "sha256-aYcodhWPcqIHh2lLDWeoq+irtg7qkWLLLK30gjQJZc8=",
			"mainSRI":              "sha256-bjpcMy7w3aaX8Cjuyv5hPE9FlkJRys0kxooPRtbGd8c=",
			"polyfillsSRI":         "sha256-+hQzPqfWEkAfOfKytrW7hLceq0mUR3pHXn+UzwhrWQ0=",
			"runtimeSRI":           "sha256-2SHK1nFbucnnM02VXrl4CAKDYQbJEF9HVZstRkVbkJM=",
			"options":              string(jsonOptions),
		})
		if err != nil {
			panic(err)
		}
	}
}
