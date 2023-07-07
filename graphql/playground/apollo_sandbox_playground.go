package playground

import (
	"html/template"
	"net/http"
)

var apolloSandboxPage = template.Must(template.New("ApolloSandbox").Parse(`<!doctype html>
<html>

<head>
  <meta charset="utf-8">
  <title>{{.title}}</title>
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <link rel="icon" href="https://embeddable-sandbox.cdn.apollographql.com/_latest/public/assets/favicon-dark.png">
	<style>
	body {
		margin: 0;
		overflow: hidden;
	}
</style>
</head>

<body>
  <div style="width: 100vw; height: 100vh;" id='embedded-sandbox'></div>
  <!-- NOTE: New version available at https://embeddable-sandbox.cdn.apollographql.com/ -->
  <script rel="preload" as="script" crossorigin="anonymous" integrity="{{.mainSRI}}" type="text/javascript" src="https://embeddable-sandbox.cdn.apollographql.com/7212121cad97028b007e974956dc951ce89d683c/embeddable-sandbox.umd.production.min.js"></script>
  <script>
{{- if .endpointIsAbsolute}}
	const url = {{.endpoint}};
{{- else}}
	const url = location.protocol + '//' + location.host + {{.endpoint}};
{{- end}}
	<!-- See https://www.apollographql.com/docs/graphos/explorer/sandbox/#options -->
  new window.EmbeddedSandbox({
    target: '#embedded-sandbox',
    initialEndpoint: url,
		persistExplorerState: true,
		initialState: {
			includeCookies: true,
			pollForSchemaUpdates: false,
		}
  });
  </script>
</body>

</html>`))

// ApolloSandboxHandler responsible for setting up the altair playground
func ApolloSandboxHandler(title, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := apolloSandboxPage.Execute(w, map[string]interface{}{
			"title":              title,
			"endpoint":           endpoint,
			"endpointIsAbsolute": endpointHasScheme(endpoint),
			"mainSRI":            "sha256-ldbSJ7EovavF815TfCN50qKB9AMvzskb9xiG71bmg2I=",
		})
		if err != nil {
			panic(err)
		}
	}
}
