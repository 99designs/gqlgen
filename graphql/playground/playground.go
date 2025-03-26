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
		integrity="{{.ReactSRI}}"
		crossorigin="anonymous"
	></script>
	<script
		src="https://cdn.jsdelivr.net/npm/react-dom@18.2.0/umd/react-dom.production.min.js"
		integrity="{{.ReactDOMSRI}}"
		crossorigin="anonymous"
	></script>
	<link
		rel="stylesheet"
		href="https://cdn.jsdelivr.net/npm/graphiql@{{.Version}}/graphiql.min.css"
		integrity="{{.CssSRI}}"
		crossorigin="anonymous"
	/>
{{- if .EnablePluginExplorer}}
	<link
		rel="stylesheet"
		href="https://cdn.jsdelivr.net/npm/@graphiql/plugin-explorer@{{.PluginExplorerVersion}}/dist/style.css"
		integrity="{{.PluginExplorerCssSRI}}"
		crossorigin="anonymous"
	/>
{{- end}}
  </head>
  <body>
    <div id="graphiql">Loading...</div>

	<script
		src="https://cdn.jsdelivr.net/npm/graphiql@{{.Version}}/graphiql.min.js"
		integrity="{{.JsSRI}}"
		crossorigin="anonymous"
	></script>
{{- if .EnablePluginExplorer}}
	<script
		src="https://cdn.jsdelivr.net/npm/@graphiql/plugin-explorer@{{.PluginExplorerVersion}}/dist/index.umd.js"
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
	Title                 string
	StoragePrefix         string
	Endpoint              string
	FetcherHeaders        map[string]string
	UiHeaders             map[string]string
	EndpointIsAbsolute    bool
	SubscriptionEndpoint  string
	Version               string
	EnablePluginExplorer  bool
	PluginExplorerVersion string
	// https://www.jsdelivr.com/package/npm/@graphiql/plugin-explorer?tab=files
	PluginExplorerCssSRI string
	PluginExplorerJsSRI  string
	// https://www.jsdelivr.com/package/npm/graphiql?tab=files
	CssSRI string
	JsSRI  string
	// https://www.jsdelivr.com/package/npm/react?tab=files
	ReactSRI string
	// https://www.jsdelivr.com/package/npm/react-dom?tab=files
	ReactDOMSRI string
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
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		var data = GraphiqlConfig{
			Title:                 title,
			Endpoint:              endpoint,
			EndpointIsAbsolute:    endpointHasScheme(endpoint),
			SubscriptionEndpoint:  getSubscriptionEndpoint(endpoint),
			Version:               "3.7.0",
			CssSRI:                "sha256-Dbkv2LUWis+0H4Z+IzxLBxM2ka1J133lSjqqtSu49o8=",
			JsSRI:                 "sha256-qsScAZytFdTAEOM8REpljROHu8DvdvxXBK7xhoq5XD0=",
			ReactSRI:              "sha256-S0lp+k7zWUMk2ixteM6HZvu8L9Eh//OVrt+ZfbCpmgY=",
			ReactDOMSRI:           "sha256-IXWO0ITNDjfnNXIu5POVfqlgYoop36bDzhodR6LW5Pc=",
			PluginExplorerVersion: "3.2.5",
			PluginExplorerCssSRI:  "sha256-+fdus37Qf3cEIKiD3VvTvgMdc8qOAT1NGUKEevz5l6k=",
			PluginExplorerJsSRI:   "sha256-minamf9GZIDrlzoMXDvU55DKk6DC5D6pNctIDWFMxS0=",
		}
		for _, opt := range opts {
			opt(&data)
		}
		err := page.Execute(w, data)
		if err != nil {
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
	return Handler(title, endpoint, WithGraphiqlFetcherHeaders(fetcherHeaders), WithGraphiqlUiHeaders(uiHeaders))
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
