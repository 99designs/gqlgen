import {describe, expect, it} from "vitest";
import {InMemoryCache, ApolloClient, HttpLink, gql} from '@apollo/client/core';

const uri = process.env.SERVER_URL || 'http://localhost:4000/';

const client = new ApolloClient({
    link: new HttpLink({uri}),
    cache: new InMemoryCache(),
});

describe('Json', () => {
    it('can join across services', async () => {
        console.log(uri)
        let res = await client.query({
            query: gql(`query {
                me {
                    username
                    reviews {
                        body
                        product {
                            name
                            upc
                        }
                    }
                }
            }`),
        });

        expect(res.data).toEqual({
            "me": {
                "__typename": "User",
                "username": "Me",
                "reviews": [
                    {
                        "__typename": "Review",
                        "body": "A highly effective form of birth control.",
                        "product": {
                            "__typename": "Product",
                            "name": "Trilby",
                            "upc": "top-1"
                        }
                    },
                    {
                        "__typename": "Review",
                        "body": "Fedoras are one of the most fashionable hats around and can look great with a variety of outfits.",
                        "product": {
                            "__typename": "Product",
                            "name": "Fedora",
                            "upc": "top-2"
                        }
                    }
                ]
            }
        });
        expect(res.error).toBeUndefined();
    });
});

