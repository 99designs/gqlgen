package playground

import (
	"html/template"
	"net/http"
	"net/url"
)

var page = template.Must(template.New("graphiql").Parse(`<!DOCTYPE html>
<html>
  <head>
  	<meta charset="utf-8">
  	<title>{{.title}}</title>
	<style>
		body {
			height: 100%;
			margin: 0;
			width: 100%;
			overflow: hidden;
		}

		#graphiql {
			height: 100vh;
		}
	</style>
	<script
		src="https://cdn.jsdelivr.net/npm/react@18.2.0/umd/react.production.min.js"
		integrity="{{.reactSRI}}"
		crossorigin="anonymous"
	></script>
	<script
		src="https://cdn.jsdelivr.net/npm/react-dom@18.2.0/umd/react-dom.production.min.js"
		integrity="{{.reactDOMSRI}}"
		crossorigin="anonymous"
	></script>
    <link
		rel="stylesheet"
		href="https://cdn.jsdelivr.net/npm/graphiql@{{.version}}/graphiql.min.css"
		integrity="{{.cssSRI}}"
		crossorigin="anonymous"
	/>
  </head>
  <body>
    <div id="graphiql">Loading...</div>

	<script
		src="https://cdn.jsdelivr.net/npm/graphiql@{{.version}}/graphiql.min.js"
		integrity="{{.jsSRI}}"
		crossorigin="anonymous"
	></script>

    <script>
{{- if .endpointIsAbsolute}}
      const url = {{.endpoint}};
      const subscriptionUrl = {{.subscriptionEndpoint}};
{{- else}}
      const url = location.protocol + '//' + location.host + {{.endpoint}};
      const wsProto = location.protocol == 'https:' ? 'wss:' : 'ws:';
      const subscriptionUrl = wsProto + '//' + location.host + {{.endpoint}};
{{- end}}
{{- if .fetcherHeaders}}
      const fetcherHeaders = {{.fetcherHeaders}};
{{- else}}
      const fetcherHeaders = undefined;
{{- end}}
{{- if .uiHeaders}}
      const uiHeaders = {{.uiHeaders}};
{{- else}}
      const uiHeaders = undefined;
{{- end}}

      const fetcher = GraphiQL.createFetcher({ url, subscriptionUrl, headers: fetcherHeaders });
      ReactDOM.render(
        React.createElement(GraphiQL, {
          fetcher: fetcher,
          isHeadersEditorEnabled: true,
          shouldPersistHeaders: true,
		  headers: JSON.stringify(uiHeaders, null, 2)
        }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
`))

// Handler responsible for setting up the playground
func Handler(title, endpoint string) http.HandlerFunc {
	return HandlerWithHeaders(title, endpoint, nil, nil)
}

// HandlerWithHeaders sets up the playground.
// fetcherHeaders are used by the playground's fetcher instance and will not be visible in the UI.
// uiHeaders are default headers that will show up in the UI headers editor.
func HandlerWithHeaders(title, endpoint string, fetcherHeaders, uiHeaders map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		err := page.Execute(w, map[string]any{
			"title":                title,
			"endpoint":             endpoint,
			"fetcherHeaders":       fetcherHeaders,
			"uiHeaders":            uiHeaders,
			"endpointIsAbsolute":   endpointHasScheme(endpoint),
			"subscriptionEndpoint": getSubscriptionEndpoint(endpoint),
			"version":              "3.0.6",
			"cssSRI":               "sha256-wTzfn13a+pLMB5rMeysPPR1hO7x0SwSeQI+cnw7VdbE=",
			"jsSRI":                "sha256-eNxH+Ah7Z9up9aJYTQycgyNuy953zYZwE9Rqf5rH+r4=",
			"reactSRI":             "sha256-S0lp+k7zWUMk2ixteM6HZvu8L9Eh//OVrt+ZfbCpmgY=",
			"reactDOMSRI":          "sha256-IXWO0ITNDjfnNXIu5POVfqlgYoop36bDzhodR6LW5Pc=",
		})
		if err != nil {
			panic(err)
		}
	}
}

// endpointHasScheme checks if the endpoint has a scheme.
func endpointHasScheme(endpoint string) bool {
	u, err := url.Parse(endpoint)
	return err == nil && u.Scheme != ""
}

// getSubscriptionEndpoint returns the subscription endpoint for the given
// endpoint if it is parsable as a URL, or an empty string.
func getSubscriptionEndpoint(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}

	return u.String()
}
