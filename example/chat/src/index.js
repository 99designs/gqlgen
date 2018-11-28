import React from 'react';
import ReactDOM from 'react-dom';
import { ApolloProvider } from 'react-apollo';
import ApolloClient from 'apollo-client';
import App from './App';
import { InMemoryCache } from 'apollo-cache-inmemory';
import { split } from 'apollo-link';
import { HttpLink } from 'apollo-link-http';
import { WebSocketLink } from 'apollo-link-ws';
import { getMainDefinition } from 'apollo-utilities';

const wsLink = new WebSocketLink({
    uri: `ws://localhost:8085/query`,
    options: {
        reconnect: true
    }
});

const httpLink = new HttpLink({ uri: 'http://localhost:8085/query' });


// depending on what kind of operation is being sent
const link = split(
    // split based on operation type
    ({ query }) => {
        const { kind, operation } = getMainDefinition(query);
        return kind === 'OperationDefinition' && operation === 'subscription';
    },
    wsLink,
    httpLink,
);

const apolloClient = new ApolloClient({
    link: link,
    cache: new InMemoryCache(),
});

if (module.hot) {
    module.hot.accept('./App', () => {
        const NextApp = require('./App').default;
        render(<NextApp/>);
    })
}

function render(component) {
    ReactDOM.render(<ApolloProvider client={apolloClient}>
        {component}
    </ApolloProvider>, document.getElementById('root'));
}

render(<App />);
