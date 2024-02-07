import { HttpLink, InMemoryCache, ApolloClient} from '@apollo/client';
import fetch from 'cross-fetch';
const gql = import('graphql-tag');

var uri = process.env.SERVER_URL || 'http://localhost:4000/';

const client = new ApolloClient({
    uri: new HttpLink({uri, fetch}),
    cache: new InMemoryCache(),
});

describe('Json', () => {
    it('can join across services', async () => {
        let res = await client.query({
            query: gql`query {
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
            }`,
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
        expect(res.errors).toBe(undefined);
    });
});

