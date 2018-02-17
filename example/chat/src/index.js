import React from 'react';
import ReactDOM from 'react-dom';
import { ApolloProvider } from 'react-apollo';
import ApolloClient from 'apollo-client';
import App from './App';
import { SubscriptionClient } from 'subscriptions-transport-ws';
import { InMemoryCache } from 'apollo-cache-inmemory';


const client = new SubscriptionClient('ws://localhost:8085/query', {
    reconnect: true,
});

const apolloClient = new ApolloClient({
    link: client,
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
