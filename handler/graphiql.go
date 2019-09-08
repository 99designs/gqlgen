package handler

import (
	"html/template"
	"net/http"
)

var graphiqlPage = template.Must(template.New("graphiql").Parse(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8"/>
	<script crossorigin src="//unpkg.com/react@{{.reactVersion}}/umd/react.production.min.js"></script>
	<script crossorigin src="//unpkg.com/react-dom@{{.reactVersion}}/umd/react-dom.production.min.js"></script>
	<!--
	<script src="https://unpkg.com/react@{{.reactVersion}}/umd/react.development.js" crossorigin></script>
	<script src="https://unpkg.com/react-dom@{{.reactVersion}}/umd/react-dom.development.js" crossorigin></script>
	-->
	<script crossorigin src="//unpkg.com/graphiql-with-extensions@{{.gqlExplorerVersion}}/graphiqlWithExtensions.min.js"></script>

	<meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
	<link rel="stylesheet" href="//unpkg.com/graphiql@{{.gqlVersion}}/graphiql.css" 
		crossorigin ></link>
	<title>{{.title}}</title>
	<style type="text/css">
	html { font-family: "Open Sans", sans-serif; }
	body {
		margin: 0; padding: 0; height: 100vh;
	}
	#root {
		height: 100vh;
	}
	</style>
</head>
<body>
<div id="root"></div>
<script type="text/javascript">
window.addEventListener('load', function (event) {
	const e = React.createElement;
	const gqil = GraphiQLWithExtensions.GraphiQLWithExtensions;

	function graphQLFetcher(graphQLParams) {
		const endpoint = location.protocol + '//' + location.host + '{{.endpoint}}'
		return fetch(endpoint, {
		  method: 'post',
		  headers: { 
			  'Accept': 'application/json',
			  'Content-Type': 'application/json'
		  },
		  body: JSON.stringify(graphQLParams),
		})
		.then(response => response.json());
	}


	ReactDOM.render(
		e(gqil, {
			fetcher: graphQLFetcher
		}),
		document.getElementById('root'),
		null
	)
})
</script>
</body>
</html>
`))

// GraphiQL creates a handlerFunc that provides GraphiQL for the endpoint
// This particular version has a built-in explorer from Onegraph that makes query writing very easy
func GraphiQL(title string, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		err := graphiqlPage.Execute(w, map[string]string{
			"title":              title,
			"endpoint":           endpoint,
			"gqlExplorerVersion": "0.14.0",
			"gqlVersion":         "0.14.2",
			"reactVersion":       "16",
		})

		if err != nil {
			panic(err)
		}
	}
}
