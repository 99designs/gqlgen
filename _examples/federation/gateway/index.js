const { ApolloServer } = import('apollo-server');
const { ApolloGateway } = import("@apollo/gateway");

const gateway = new ApolloGateway({
    serviceList: [
        { name: 'accounts', url: 'http://localhost:4001/query' },
        { name: 'products', url: 'http://localhost:4002/query' },
        { name: 'reviews', url: 'http://localhost:4003/query' }
    ],
});

const server = new ApolloServer({
    gateway,

    subscriptions: false,
});

server.listen().then(({ url }) => {
    console.log(`🚀 Server ready at ${url}`);
});
