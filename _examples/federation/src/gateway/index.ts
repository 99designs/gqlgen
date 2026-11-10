import {ApolloServer} from '@apollo/server';
import {startStandaloneServer} from '@apollo/server/standalone';
import {ApolloGateway, IntrospectAndCompose} from '@apollo/gateway';

const gateway = new ApolloGateway({
    supergraphSdl: new IntrospectAndCompose({
        subgraphs: [
            {name: 'accounts', url: 'http://localhost:4001/query'},
            {name: 'products', url: 'http://localhost:4002/query'},
            {name: 'reviews', url: 'http://localhost:4003/query'}
        ],
    }),
});

const server = new ApolloServer({gateway});

// Note the top-level `await`!
const {url} = await startStandaloneServer(server);
console.log(`🚀  Server ready at ${url}`);

// The Federation v1 subgraphs in v1/, served by `go run ./v1`.
const gatewayV1 = new ApolloGateway({
    supergraphSdl: new IntrospectAndCompose({
        subgraphs: [
            {name: 'accounts', url: 'http://localhost:4011/query'},
            {name: 'content', url: 'http://localhost:4012/query'},
            {name: 'progress', url: 'http://localhost:4013/query'}
        ],
    }),
});

const serverV1 = new ApolloServer({gateway: gatewayV1});
const {url: urlV1} = await startStandaloneServer(serverV1, {listen: {port: 4010}});
console.log(`🚀  Federation v1 server ready at ${urlV1}`);
