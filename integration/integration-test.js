import {InMemoryCache} from "apollo-cache-inmemory";
import {HttpLink} from "apollo-link-http";
import {ApolloClient} from "apollo-client";
import fetch from "node-fetch";
import gql from 'graphql-tag';

var uri = process.env.SERVER_URL || 'http://localhost:8080/query';

const client = new ApolloClient({
    link: new HttpLink({uri, fetch}),
    cache: new InMemoryCache(),
    defaultOptions: {
        watchQuery: {
            fetchPolicy: 'network-only',
            errorPolicy: 'ignore',
        },
        query: {
            fetchPolicy: 'network-only',
            errorPolicy: 'all',
        },
    },
});

describe('Json', () => {
    it('should follow json escaping rules', async () => {
        let res = await client.query({
            query: gql`{ jsonEncoding }`,
        });

        expect(res.data.jsonEncoding).toBe("󾓭");
        expect(res.errors).toBe(undefined);
    });
});

describe('Input defaults', () => {
    it('should pass default values to resolver', async () => {
        let res = await client.query({
            query: gql`{ date(filter:{value: "asdf"}) }`,
        });

        expect(res.data.date).toBe(true);
        expect(res.errors).toBe(undefined);
    });
});

describe('Errors', () => {
    it('should respond with correct paths', async () => {
        let res = await client.query({
            query: gql`{ path { cc:child { error } } }`,
        });

        expect(res.errors[0].path).toEqual(['path', 0, 'cc', 'error']);
        expect(res.errors[1].path).toEqual(['path', 1, 'cc', 'error']);
        expect(res.errors[2].path).toEqual(['path', 2, 'cc', 'error']);
        expect(res.errors[3].path).toEqual(['path', 3, 'cc', 'error']);
    });

    it('should use the error presenter for custom errors', async () => {
        let res = await client.query({
            query: gql`{ error(type: CUSTOM) }`,
        });

        expect(res.errors[0].message).toEqual('User message');
    });

    it('should pass through for other errors', async () => {
        let res = await client.query({
            query: gql`{ error(type: NORMAL) }`,
        });

        expect(res.errors[0].message).toEqual('normal error');
    });
});
