import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
    overwrite: true,
    schema: process.env.VITE_SERVER_URL ?? 'http://localhost:8080/query',
    documents: 'src/**/*.graphql',
    generates: {
        'src/generated/': {
            preset: 'client-preset'
        },
        'src/generated/schema-fetched.graphql': {
            plugins: ['schema-ast'],
        },
        'src/generated/schema-introspection.json': {
            plugins: ['introspection'],
        }
    },
};

export default config;
