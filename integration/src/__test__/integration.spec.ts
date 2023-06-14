import {afterAll, describe, expect, it} from 'vitest'
import {ApolloClient, ApolloLink, FetchResult, HttpLink, InMemoryCache, NormalizedCacheObject, Observable, Operation} from "@apollo/client/core";
import {print} from 'graphql';
import {GraphQLWsLink} from "@apollo/client/link/subscriptions";
import {WebSocket} from 'ws';
import {createClient as createClientWS} from "graphql-ws";
import {Client as ClientSSE, ClientOptions as ClientOptionsSSE, createClient as createClientSSE} from 'graphql-sse';
import {CoercionDocument, ComplexityDocument, DateDocument, ErrorDocument, ErrorType, JsonEncodingDocument, PathDocument} from '../generated/graphql.ts';

const uri = process.env.VITE_SERVER_URL || 'http://localhost:8080/query';

function test(client: ApolloClient<NormalizedCacheObject>) {
    describe('Json', () => {
        it('should follow json escaping rules', async () => {
            let res = await client.query({
                query: JsonEncodingDocument,
            });

            expect(res.data.jsonEncoding).toBe("󾓭");
            expect(res.errors).toBe(undefined);

            return null;
        });
    });

    describe('Input defaults', () => {
        it('should pass default values to resolver', async () => {
            let res = await client.query({
                query: DateDocument,
                variables: {
                    filter: {
                        value: "asdf"
                    }
                }
            });

            expect(res.data.date).toBe(true);
            expect(res.errors).toBe(undefined);
            return null;
        });
    });

    describe('Complexity', () => {
        it('should fail when complexity is too high', async () => {
            let res = await client.query({
                query: ComplexityDocument,
                variables: {
                    value: 2000,
                }
            });

            expect(res.errors).toBeDefined()
            if (res.errors) {
                expect(res.errors[0].message).toBe("operation has complexity 2000, which exceeds the limit of 1000");
            }
            return null;
        });


        it('should succeed when complexity is not too high', async () => {
            let res = await client.query({
                query: ComplexityDocument,
                variables: {
                    value: 1000,
                }
            });

            expect(res.data.complexity).toBe(true);
            expect(res.errors).toBe(undefined);
            return null;
        });
    });

    describe('List Coercion', () => {

        it('should succeed when nested single values are passed', async () => {
            let res = await client.query({
                query: CoercionDocument,
                variables: {
                    value: {
                        enumVal: ErrorType.Custom,
                        strVal: "test",
                        intVal: 1,
                    }
                },
            });

            expect(res.data.coercion).toBe(true);
            return null;
        });

        it('should succeed when nested array of values are passed', async () => {
            let res = await client.query({
                query: CoercionDocument,
                variables: {
                    value: {
                        enumVal: [ErrorType.Custom],
                        strVal: ["test"],
                        intVal: [1],
                    }
                },
            });

            expect(res.data.coercion).toBe(true);
            return null;
        });

        it('should succeed when single value is passed', async () => {
            let res = await client.query({
                query: CoercionDocument,
                variables: {
                    value: {
                        enumVal: ErrorType.Custom,
                    }
                },
            });

            expect(res.data.coercion).toBe(true);
            return null;
        });

        it('should succeed when single scalar value is passed', async () => {
            let res = await client.query({
                query: CoercionDocument,
                variables: {
                    value: [{
                        scalarVal: {
                            key: 'someValue'
                        }
                    }]
                }
            });

            expect(res.data.coercion).toBe(true);
            return null;
        });

        it('should succeed when multiple values are passed', async () => {
            let res = await client.query({
                query: CoercionDocument,
                variables: {
                    value: [{
                        enumVal: [ErrorType.Custom,ErrorType.Normal]
                    }]
                }
            });

            expect(res.data.coercion).toBe(true);
            return null;
        });

    });

    describe('Errors', () => {
        it('should respond with correct paths', async () => {
            let res = await client.query({
                query: PathDocument,
            });

            expect(res.errors).toBeDefined()
            if (res.errors) {
                expect(res.errors[0].path).toEqual(['path', 0, 'cc', 'error']);
                expect(res.errors[1].path).toEqual(['path', 1, 'cc', 'error']);
                expect(res.errors[2].path).toEqual(['path', 2, 'cc', 'error']);
                expect(res.errors[3].path).toEqual(['path', 3, 'cc', 'error']);
            }
            return null;
        });

        it('should use the error presenter for custom errors', async () => {
            let res = await client.query({
                query: ErrorDocument,
                variables: {
                    type: ErrorType.Custom
                }
            });

            expect(res.errors).toBeDefined()
            if (res.errors) {
                expect(res.errors[0].message).toEqual('User message');
            }
            return null;
        });

        it('should pass through for other errors', async () => {
            let res = await client.query({
                query: ErrorDocument,
                variables: {
                    type: ErrorType.Normal
                }
            });

            expect(res.errors).toBeDefined()
            if (res.errors) {
                expect(res.errors[0]?.message).toEqual('normal error');
            }
            return null;
        });
    });
}

describe('HTTP client', () => {
    const client = new ApolloClient({
        link: new HttpLink({
            uri,
            fetch,
        }),
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

    test(client);

    afterAll(() => {
        client.stop();
    });
});

describe('Websocket client', () => {
    const client = new ApolloClient({
        link: new GraphQLWsLink(
            createClientWS({
                url: uri.replace('http://', 'ws://').replace('https://', 'wss://'),
                webSocketImpl: WebSocket,
            }),
        ),
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

    test(client);

    afterAll(() => {
        client.stop();
    });
});

describe('SSE client', () => {
    class SSELink extends ApolloLink {
        private client: ClientSSE;

        constructor(options: ClientOptionsSSE) {
            super();
            this.client = createClientSSE(options);
        }

        public request(operation: Operation): Observable<FetchResult> {
            return new Observable((sink) => {
                return this.client.subscribe<FetchResult>(
                    {...operation, query: print(operation.query)},
                    {
                        next: sink.next.bind(sink),
                        complete: sink.complete.bind(sink),
                        error: sink.error.bind(sink),
                    },
                );
            });
        }
    }

    const client = new ApolloClient({
        link: new SSELink({
            url: uri,
        }),
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

    test(client);

    afterAll(() => {
        client.stop();
    });
});
