/* eslint-disable */
import * as types from './graphql';
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';

/**
 * Map of all GraphQL operations in the project.
 *
 * This map has several performance disadvantages:
 * 1. It is not tree-shakeable, so it will include all operations in the project.
 * 2. It is not minifiable, so the string of a GraphQL query will be multiple times inside the bundle.
 * 3. It does not support dead code elimination, so it will add unused operations.
 *
 * Therefore it is highly recommended to use the babel or swc plugin for production.
 */
const documents = {
    "query coercion($value: [ListCoercion!]) {\n  coercion(value: $value)\n}": types.CoercionDocument,
    "query complexity($value: Int!) {\n  complexity(value: $value)\n}": types.ComplexityDocument,
    "query date($filter: DateFilter!) {\n  date(filter: $filter)\n}": types.DateDocument,
    "query error($type: ErrorType) {\n  error(type: $type)\n}": types.ErrorDocument,
    "query jsonEncoding {\n  jsonEncoding\n}": types.JsonEncodingDocument,
    "query path {\n  path {\n    cc: child {\n      error\n    }\n  }\n}": types.PathDocument,
    "query viewer {\n  viewer {\n    user {\n      name\n      phoneNumber\n      query {\n        jsonEncoding\n      }\n      ...userFragment @defer\n    }\n  }\n}\n\nfragment userFragment on User {\n  likes\n}": types.ViewerDocument,
};

/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 *
 *
 * @example
 * ```ts
 * const query = graphql(`query GetUser($id: ID!) { user(id: $id) { name } }`);
 * ```
 *
 * The query argument is unknown!
 * Please regenerate the types.
 */
export function graphql(source: string): unknown;

/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "query coercion($value: [ListCoercion!]) {\n  coercion(value: $value)\n}"): (typeof documents)["query coercion($value: [ListCoercion!]) {\n  coercion(value: $value)\n}"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "query complexity($value: Int!) {\n  complexity(value: $value)\n}"): (typeof documents)["query complexity($value: Int!) {\n  complexity(value: $value)\n}"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "query date($filter: DateFilter!) {\n  date(filter: $filter)\n}"): (typeof documents)["query date($filter: DateFilter!) {\n  date(filter: $filter)\n}"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "query error($type: ErrorType) {\n  error(type: $type)\n}"): (typeof documents)["query error($type: ErrorType) {\n  error(type: $type)\n}"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "query jsonEncoding {\n  jsonEncoding\n}"): (typeof documents)["query jsonEncoding {\n  jsonEncoding\n}"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "query path {\n  path {\n    cc: child {\n      error\n    }\n  }\n}"): (typeof documents)["query path {\n  path {\n    cc: child {\n      error\n    }\n  }\n}"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "query viewer {\n  viewer {\n    user {\n      name\n      phoneNumber\n      query {\n        jsonEncoding\n      }\n      ...userFragment @defer\n    }\n  }\n}\n\nfragment userFragment on User {\n  likes\n}"): (typeof documents)["query viewer {\n  viewer {\n    user {\n      name\n      phoneNumber\n      query {\n        jsonEncoding\n      }\n      ...userFragment @defer\n    }\n  }\n}\n\nfragment userFragment on User {\n  likes\n}"];

export function graphql(source: string) {
  return (documents as any)[source] ?? {};
}

export type DocumentType<TDocumentNode extends DocumentNode<any, any>> = TDocumentNode extends DocumentNode<  infer TType,  any>  ? TType  : never;