import React from 'react';
import ReactDOM from 'react-dom';
import {
    ApolloClient,
    ApolloProvider,
    HttpLink,
    split,
    InMemoryCache,
} from '@apollo/client';
import { WebSocketLink as ApolloWebSocketLink} from '@apollo/client/link/ws';
import { getMainDefinition } from 'apollo-utilities';
import { App } from './App';
import { WebSocketLink as GraphQLWSWebSocketLink } from './graphql-ws'
import { SSELink } from './graphql-sse';

let subscriptionLink;
if (process.env.REACT_APP_SSE_PROTOCOL) {
  subscriptionLink = new SSELink({ url: 'http://localhost:8085/query' });
} else if (process.env.REACT_APP_WS_PROTOCOL === 'graphql-transport-ws') {
    subscriptionLink = new GraphQLWSWebSocketLink({
        url: `ws://localhost:8085/query`
    });
} else {
    subscriptionLink = new ApolloWebSocketLink({
        uri: `ws://localhost:8085/query`,
        options: {
            reconnect: true
        }
    });
}

const httpLink = new HttpLink({ uri: 'http://localhost:8085/query' });


// depending on what kind of operation is being sent
const link = split(
    // split based on operation type
    ({ query }) => {
        const { kind, operation } = getMainDefinition(query);
        return kind === 'OperationDefinition' && operation === 'subscription';
    },
    subscriptionLink,
    httpLink,
);

const apolloClient = new ApolloClient({
    link: link,
    cache: new InMemoryCache(),
});

if (module.hot) {
    module.hot.accept('./App', () => {
        const NextApp = import('./App').default;
        render(<NextApp/>);
    })
}

function render(component) {
    ReactDOM.render(<ApolloProvider client={apolloClient}>
        {component}
    </ApolloProvider>, document.getElementById('root'));
}

render(<App />);
