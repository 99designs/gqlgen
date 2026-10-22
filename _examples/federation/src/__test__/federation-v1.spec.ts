import {describe, expect, it} from "vitest";
import {readFileSync} from "node:fs";
import {parse} from "graphql";
import {composeServices} from "@apollo/composition";

// The gateway that src/gateway/index.ts starts for the Federation v1
// subgraphs in v1/.
const uri = process.env.SERVER_V1_URL || 'http://localhost:4010/';

async function post(query: string, variables?: Record<string, unknown>) {
    const res = await fetch(uri, {
        method: 'POST',
        headers: {'content-type': 'application/json'},
        body: JSON.stringify({query, variables}),
    });
    expect(res.status).toBe(200);
    return res.json();
}

describe('Federation v1', () => {
    it('composes the subgraph schema files', () => {
        const result = composeServices(['accounts', 'content', 'progress'].map(name => ({
            name,
            url: `http://${name}.invalid/query`,
            typeDefs: parse(readFileSync(`v1/${name}/graph/schema.graphqls`, 'utf8')),
        })));
        expect(result.errors?.map(String)).toBeUndefined();
    });

    it.each([
        {
            name: 'entities across subgraphs with @requires and @provides',
            query: `query MyProgress {
                me {
                    id
                    username
                    greeting
                    progress(since: "2024-01-01") {
                        percent
                        completedOn
                        video { id title durationSeconds }
                    }
                }
            }`,
            want: {
                me: {
                    id: '1',
                    username: 'sal',
                    greeting: 'Hi, sal!',
                    progress: [{
                        percent: 1,
                        completedOn: '2024-03-04',
                        video: {id: 'video-1', title: 'Adding fractions', durationSeconds: 300},
                    }],
                },
            },
        },
        {
            name: 'a deprecated argument',
            query: `query DeprecatedArgument { me { avatarURL(px: 64) } }`,
            want: {me: {avatarURL: 'https://avatars.example.com/1?size=64'}},
        },
        {
            name: 'interfaces, unions, fragments and @include',
            query: `query Search($query: String!, $withArticles: Boolean!) {
                search(query: $query, kinds: [VIDEO, ARTICLE], limit: 5) {
                    __typename
                    ... on Video { id title }
                    ... on Article @include(if: $withArticles) { wordCount }
                }
                content(id: "video-1") { ...ContentFields }
            }
            fragment ContentFields on Content {
                id
                title
                ... on Video { publishedOn }
            }`,
            variables: {query: 'fractions', withArticles: true},
            want: {
                search: [
                    {__typename: 'Video', id: 'video-1', title: 'Adding fractions'},
                    {__typename: 'Article', wordCount: 1200},
                ],
                content: {id: 'video-1', title: 'Adding fractions', publishedOn: '2024-01-02'},
            },
        },
        {
            name: 'a mutation with a deprecated input field',
            query: `mutation CreateStudents($input: CreateStudentsInput!) {
                createStudents(input: $input, notify: true) {
                    id
                    username
                    progress { percent }
                }
            }`,
            variables: {input: {usernames: ['alex'], listKey: 'period-1'}},
            want: {
                createStudents: [{id: 'period-1-alex', username: 'alex', progress: [{percent: 1}]}],
            },
        },
        {
            name: 'introspection of deprecated input fields',
            query: `{
                __type(name: "CreateStudentsInput") {
                    inputFields(includeDeprecated: true) { name isDeprecated }
                }
            }`,
            want: {
                __type: {
                    inputFields: [
                        {name: 'usernames', isDeprecated: false},
                        {name: 'classroomID', isDeprecated: false},
                        {name: 'listKey', isDeprecated: true},
                    ],
                },
            },
        },
    ])('resolves $name', async ({query, variables, want}) => {
        const res = await post(query, variables);
        expect(res.errors).toBeUndefined();
        expect(res.data).toEqual(want);
    });
});
