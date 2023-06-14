import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
    overwrite: true,
    schema: process.env.VITE_SERVER_URL ?? 'http://localhost:8080/query',
    documents: 'src/**/*.graphql',
    generates: {
        'src/generated/graphql.ts': {
            plugins: ['typescript', 'typescript-operations', 'typed-document-node'],
        },
        'src/generated/schema-fetched.graphql': {
            plugins: ['schema-ast'],
        },
    },
};

export default config;
