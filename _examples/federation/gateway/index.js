import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { ApolloGateway,IntrospectAndCompose } from '@apollo/gateway';

const gateway = new ApolloGateway({
    supergraphSdl: new IntrospectAndCompose({
        subgraphs: [
            { name: 'accounts', url: 'http://localhost:4001/query' },
            { name: 'products', url: 'http://localhost:4002/query' },
            { name: 'reviews', url: 'http://localhost:4003/query' }
        ],
    }),
});

const server = new ApolloServer({
    gateway,
    subscriptions: false,
});

// Note the top-level `await`!
const { url } = await startStandaloneServer(server);
console.log(`🚀  Server ready at ${url}`);
