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
  	<title>{{.Title}}</title>
	<style>
		body {
			margin: 0;
		}

		#graphiql {
			height: 100vh;
		}

		.loading {
        	height: 100%;
        	display: flex;
        	align-items: center;
        	justify-content: center;
        	font-size: 4rem;
		}
	</style>
	<script
		src="{{.ReactUrl}}"
		integrity="{{.ReactSRI}}"
		crossorigin="anonymous"
	></script>
	<script
		src="{{.ReactDOMUrl}}"
		integrity="{{.ReactDOMSRI}}"
		crossorigin="anonymous"
	></script>
	<link
		rel="stylesheet"
		href="{{.CssUrl}}"
		integrity="{{.CssSRI}}"
		crossorigin="anonymous"
	/>
{{- if .EnablePluginExplorer}}
	<link
		rel="stylesheet"
		href="{{.PluginExplorerCssUrl}}"
		integrity="{{.PluginExplorerCssSRI}}"
		crossorigin="anonymous"
	/>
{{- end}}
  </head>
  <body>
    <div id="graphiql">
		<div class="loading">Loading…</div>
	</div>

	<script
		src="{{.JsUrl}}"
		integrity="{{.JsSRI}}"
		crossorigin="anonymous"
	></script>
{{- if .EnablePluginExplorer}}
	<script
		src="{{.PluginExplorerJsUrl}}"
		integrity="{{.PluginExplorerJsSRI}}"
		crossorigin="anonymous"
	></script>
{{- end}}

    <script>
      class PrefixedStorage {
        constructor(prefix = '') {
          this.prefix = prefix;
        }

        _addPrefix(key) {
          return this.prefix + key;
        }

        _removePrefix(prefixedKey) {
          return prefixedKey.substring(this.prefix.length);
        }

        setItem(key, value) {
          const prefixedKey = this._addPrefix(key);
          localStorage.setItem(prefixedKey, value);
        }

        getItem(key) {
          const prefixedKey = this._addPrefix(key);
          return localStorage.getItem(prefixedKey);
        }

        removeItem(key) {
          const prefixedKey = this._addPrefix(key);
          localStorage.removeItem(prefixedKey);
        }

        clear() {
          const keysToRemove = [];
          for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key.startsWith(this.prefix)) {
              keysToRemove.push(key);
            }
          }
          keysToRemove.forEach(key => localStorage.removeItem(key));
        }

        get length() {
          let count = 0;
          for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key.startsWith(this.prefix)) {
              count++;
            }
          }
          return count;
        }

        key(index) {
          const keys = [];
          for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key.startsWith(this.prefix)) {
              keys.push(this._removePrefix(key));
            }
          }
          return index >= 0 && index < keys.length ? keys[index] : null;
        }
      }
{{- if .EndpointIsAbsolute}}
      const url = {{.Endpoint}};
      const subscriptionUrl = {{.SubscriptionEndpoint}};
{{- else}}
      const url = location.protocol + '//' + location.host + {{.Endpoint}};
      const wsProto = location.protocol == 'https:' ? 'wss:' : 'ws:';
      const subscriptionUrl = wsProto + '//' + location.host + {{.Endpoint}};
{{- end}}
{{- if .FetcherHeaders}}
      const fetcherHeaders = {{.FetcherHeaders}};
{{- else}}
      const fetcherHeaders = undefined;
{{- end}}
{{- if .UiHeaders}}
      const uiHeaders = {{.UiHeaders}};
{{- else}}
      const uiHeaders = undefined;
{{- end}}

      let plugins = [];
{{- if .EnablePluginExplorer}}
      plugins.push(GraphiQLPluginExplorer.explorerPlugin());
{{- end}}

      const fetcher = GraphiQL.createFetcher({ url, subscriptionUrl, headers: fetcherHeaders });
      ReactDOM.render(
        React.createElement(GraphiQL, {
          fetcher: fetcher,
          isHeadersEditorEnabled: true,
          shouldPersistHeaders: true,
		  headers: JSON.stringify(uiHeaders, null, 2),
		  plugins: plugins,
          storage: new PrefixedStorage('{{.StoragePrefix}}')
        }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
`))

type GraphiqlConfig struct {
	Title                string
	StoragePrefix        string
	Endpoint             string
	FetcherHeaders       map[string]string
	UiHeaders            map[string]string
	EndpointIsAbsolute   bool
	SubscriptionEndpoint string
	JsUrl                template.URL
	JsSRI                string
	CssUrl               template.URL
	CssSRI               string
	ReactUrl             template.URL
	ReactSRI             string
	ReactDOMUrl          template.URL
	ReactDOMSRI          string
	EnablePluginExplorer bool
	PluginExplorerJsUrl  template.URL
	PluginExplorerJsSRI  string
	PluginExplorerCssUrl template.URL
	PluginExplorerCssSRI string
}
type GraphiqlConfigOption func(*GraphiqlConfig)

func WithGraphiqlFetcherHeaders(headers map[string]string) GraphiqlConfigOption {
	return func(config *GraphiqlConfig) {
		config.FetcherHeaders = headers
	}
}

func WithGraphiqlUiHeaders(headers map[string]string) GraphiqlConfigOption {
	return func(config *GraphiqlConfig) {
		config.UiHeaders = headers
	}
}

func WithGraphiqlVersion(jsUrl, cssUrl, jsSri, cssSri string) GraphiqlConfigOption {
	return func(config *GraphiqlConfig) {
		config.JsUrl = template.URL(jsUrl)
		config.CssUrl = template.URL(cssUrl)
		config.JsSRI = jsSri
		config.CssSRI = cssSri
	}
}

func WithGraphiqlReactVersion(
	reactJsUrl, reactDomJsUrl, reactJsSri, reactDomJsSri string,
) GraphiqlConfigOption {
	return func(config *GraphiqlConfig) {
		config.ReactUrl = template.URL(reactJsUrl)
		config.ReactDOMUrl = template.URL(reactDomJsUrl)
		config.ReactSRI = reactJsSri
		config.ReactDOMSRI = reactDomJsSri
	}
}

func WithGraphiqlPluginExplorerVersion(jsUrl, cssUrl, jsSri, cssSri string) GraphiqlConfigOption {
	return func(config *GraphiqlConfig) {
		config.PluginExplorerJsUrl = template.URL(jsUrl)
		config.PluginExplorerCssUrl = template.URL(cssUrl)
		config.PluginExplorerJsSRI = jsSri
		config.PluginExplorerCssSRI = cssSri
	}
}

func WithGraphiqlEnablePluginExplorer(enable bool) GraphiqlConfigOption {
	return func(config *GraphiqlConfig) {
		config.EnablePluginExplorer = enable
	}
}

func WithStoragePrefix(prefix string) GraphiqlConfigOption {
	return func(config *GraphiqlConfig) {
		config.StoragePrefix = prefix
	}
}

// Handler responsible for setting up the playground
func Handler(title, endpoint string, opts ...GraphiqlConfigOption) http.HandlerFunc {
	data := GraphiqlConfig{
		Title:                title,
		Endpoint:             endpoint,
		EndpointIsAbsolute:   endpointHasScheme(endpoint),
		SubscriptionEndpoint: getSubscriptionEndpoint(endpoint),
		// https://www.jsdelivr.com/package/npm/graphiql?tab=files
		JsUrl:  "https://cdn.jsdelivr.net/npm/graphiql@4.1.2/graphiql.min.js",
		JsSRI:  "sha256-hnImuor1znlJkD/FOTL3jayfS/xsyNoP04abi8bFJWs=",
		CssUrl: "https://cdn.jsdelivr.net/npm/graphiql@4.1.2/graphiql.min.css",
		CssSRI: "sha256-MEh+B2NdMSpj9kexQNN3QKc8UzMrCXW/Sx/phcpuyIU=",
		// https://www.jsdelivr.com/package/npm/react?tab=files
		ReactUrl: "https://cdn.jsdelivr.net/npm/react@18.2.0/umd/react.production.min.js",
		ReactSRI: "sha256-S0lp+k7zWUMk2ixteM6HZvu8L9Eh//OVrt+ZfbCpmgY=",
		// https://www.jsdelivr.com/package/npm/react-dom?tab=files
		ReactDOMUrl: "https://cdn.jsdelivr.net/npm/react-dom@18.2.0/umd/react-dom.production.min.js",
		ReactDOMSRI: "sha256-IXWO0ITNDjfnNXIu5POVfqlgYoop36bDzhodR6LW5Pc=",
		// https://www.jsdelivr.com/package/npm/@graphiql/plugin-explorer?tab=files
		PluginExplorerJsUrl: template.URL(
			"https://cdn.jsdelivr.net/npm/@graphiql/plugin-explorer@4.0.6/dist/index.umd.js",
		),
		PluginExplorerJsSRI: "sha256-UM8sWOS0Xa9yLY85q6Clh0pF4qpxX+TOcJ41flECqBs=",
		PluginExplorerCssUrl: template.URL(
			"https://cdn.jsdelivr.net/npm/@graphiql/plugin-explorer@4.0.6/dist/style.min.css",
		),
		PluginExplorerCssSRI: "sha256-b0izygy8aEMY3fCLmtNkm9PKdE3kRD4Qjn6Q8gw5xKI=",
	}
	for _, opt := range opts {
		opt(&data)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")

		if err := page.Execute(w, data); err != nil {
			panic(err)
		}
	}
}

// HandlerWithHeaders sets up the playground.
// fetcherHeaders are used by the playground's fetcher instance and will not be visible in the UI.
// uiHeaders are default headers that will show up in the UI headers editor.
func HandlerWithHeaders(
	title, endpoint string,
	fetcherHeaders, uiHeaders map[string]string,
) http.HandlerFunc {
	return Handler(
		title,
		endpoint,
		WithGraphiqlFetcherHeaders(fetcherHeaders),
		WithGraphiqlUiHeaders(uiHeaders),
	)
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
