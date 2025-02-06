package playground

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

// NOTE: New version available at https://embeddable-sandbox.cdn.apollographql.com/ -->
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
  <script rel="preload" as="script" crossorigin="anonymous" integrity="{{.mainSRI}}" type="text/javascript" src="https://embeddable-sandbox.cdn.apollographql.com/02e2da0fccbe0240ef03d2396d6c98559bab5b06/embeddable-sandbox.umd.production.min.js"></script>
  <script>
{{- if .endpointIsAbsolute}}
	const url = {{.endpoint}};
{{- else}}
	const url = location.protocol + '//' + location.host + {{.endpoint}};
{{- end}}
	const options = JSON.parse({{.options}})
	new window.EmbeddedSandbox({
		target: '#embedded-sandbox',
		initialEndpoint: url,
		persistExplorerState: true,
		...options,
	});
  </script>
</body>
</html>`))

// ApolloSandboxHandler responsible for setting up the apollo sandbox playground
func ApolloSandboxHandler(title, endpoint string, opts ...ApolloSandboxOption) http.HandlerFunc {
	options := &apolloSandboxOptions{
		HideCookieToggle: true,
		InitialState: apolloSandboxInitialState{
			IncludeCookies:       true,
			PollForSchemaUpdates: false,
		},
	}

	for _, opt := range opts {
		opt(options)
	}

	optionsBytes, err := json.Marshal(options)
	if err != nil {
		panic(fmt.Errorf("failed to marshal apollo sandbox options: %w", err))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := apolloSandboxPage.Execute(w, map[string]any{
			"title":              title,
			"endpoint":           endpoint,
			"endpointIsAbsolute": endpointHasScheme(endpoint),
			"mainSRI":            "sha256-pYhw/8TGkZxk960PMMpDtjhw9YtKXUzGv6XQQaMJSh8=",
			"options":            string(optionsBytes),
		})
		if err != nil {
			panic(err)
		}
	}
}

// See https://www.apollographql.com/docs/graphos/explorer/sandbox/#options -->
type apolloSandboxOptions struct {
	HideCookieToggle   bool                      `json:"hideCookieToggle"`
	EndpointIsEditable bool                      `json:"endpointIsEditable"`
	InitialState       apolloSandboxInitialState `json:"initialState,omitempty"`
}

type apolloSandboxInitialState struct {
	IncludeCookies       bool           `json:"includeCookies"`
	Document             string         `json:"document,omitempty"`
	Variables            map[string]any `json:"variables,omitempty"`
	Headers              map[string]any `json:"headers,omitempty"`
	CollectionId         string         `json:"collectionId,omitempty"`
	OperationId          string         `json:"operationId,omitempty"`
	PollForSchemaUpdates bool           `json:"pollForSchemaUpdates"`
	SharedHeaders        map[string]any `json:"sharedHeaders,omitempty"`
}

type ApolloSandboxOption func(options *apolloSandboxOptions)

// WithApolloSandboxHideCookieToggle By default, the embedded Sandbox does not show the Include cookies toggle in its connection settings.
//
// Set hideCookieToggle to false to enable users of your embedded Sandbox instance to toggle the Include cookies setting.
func WithApolloSandboxHideCookieToggle(hideCookieToggle bool) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.HideCookieToggle = hideCookieToggle
	}
}

// WithApolloSandboxEndpointIsEditable By default, the embedded Sandbox has a URL input box that is editable by users.
//
// Set endpointIsEditable to false to prevent users of your embedded Sandbox instance from changing the endpoint URL.
func WithApolloSandboxEndpointIsEditable(endpointIsEditable bool) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.EndpointIsEditable = endpointIsEditable
	}
}

// WithApolloSandboxInitialStateIncludeCookies Set this value to true if you want the Sandbox to pass { credentials: 'include' } for its requests by default.
//
// If you set hideCookieToggle to false, users can override this default setting with the Include cookies toggle. (By default, the embedded Sandbox does not show the Include cookies toggle in its connection settings.)
//
// If you also pass the handleRequest option, this option is ignored.
//
// Read more about the fetch API and credentials here https://developer.mozilla.org/en-US/docs/Web/API/fetch#credentials
func WithApolloSandboxInitialStateIncludeCookies(includeCookies bool) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.InitialState.IncludeCookies = includeCookies
	}
}

// WithApolloSandboxInitialStateDocument Document operation to populate in the Sandbox's editor on load.
//
// If you omit this, the Sandbox initially loads an example query based on your schema.
func WithApolloSandboxInitialStateDocument(document string) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.InitialState.Document = document
	}
}

// WithApolloSandboxInitialStateVariables Variables containing initial variable values to populate in the Sandbox on load.
//
// If provided, these variables should apply to the initial query you provide for document.
func WithApolloSandboxInitialStateVariables(variables map[string]any) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.InitialState.Variables = variables
	}
}

// WithApolloSandboxInitialStateHeaders Headers containing initial variable values to populate in the Sandbox on load.
//
// If provided, these variables should apply to the initial query you provide for document.
func WithApolloSandboxInitialStateHeaders(headers map[string]any) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.InitialState.Headers = headers
	}
}

// WithApolloSandboxInitialStateCollectionIdAndOperationId The ID of a collection, paired with an operation ID to populate in the Sandbox on load.
//
// You can find these values from a registered graph in Studio by clicking the ... menu next to an operation in the Explorer of that graph and selecting View operation details.
func WithApolloSandboxInitialStateCollectionIdAndOperationId(collectionId, operationId string) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.InitialState.CollectionId = collectionId
		options.InitialState.OperationId = operationId
	}
}

// WithApolloSandboxInitialStatePollForSchemaUpdates If true, the embedded Sandbox periodically polls your initialEndpoint for schema updates.
//
// The default value is false.
func WithApolloSandboxInitialStatePollForSchemaUpdates(pollForSchemaUpdates bool) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.InitialState.PollForSchemaUpdates = pollForSchemaUpdates
	}
}

// WithApolloSandboxInitialStateSharedHeaders Headers that are applied by default to every operation executed by the embedded Sandbox.
//
// Users can disable the application of these headers, but they can't modify their values.
//
// The embedded Sandbox always includes these headers in its introspection queries to your initialEndpoint.
func WithApolloSandboxInitialStateSharedHeaders(sharedHeaders map[string]any) ApolloSandboxOption {
	return func(options *apolloSandboxOptions) {
		options.InitialState.SharedHeaders = sharedHeaders
	}
}
