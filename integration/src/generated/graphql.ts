import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string | number; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Map: { input: any; output: any; }
};

export enum Date_Filter_Op {
  Eq = 'EQ',
  Gt = 'GT',
  Gte = 'GTE',
  Lt = 'LT',
  Lte = 'LTE',
  Neq = 'NEQ'
}

export type DateFilter = {
  op?: InputMaybe<Date_Filter_Op>;
  timezone?: InputMaybe<Scalars['String']['input']>;
  value: Scalars['String']['input'];
};

export type Element = {
  __typename?: 'Element';
  child: Element;
  error: Scalars['Boolean']['output'];
  mismatched?: Maybe<Array<Scalars['Boolean']['output']>>;
};

export enum ErrorType {
  Custom = 'CUSTOM',
  Normal = 'NORMAL'
}

export type ListCoercion = {
  enumVal?: InputMaybe<Array<InputMaybe<ErrorType>>>;
  intVal?: InputMaybe<Array<InputMaybe<Scalars['Int']['input']>>>;
  scalarVal?: InputMaybe<Array<InputMaybe<Scalars['Map']['input']>>>;
  strVal?: InputMaybe<Array<InputMaybe<Scalars['String']['input']>>>;
};

export type Query = {
  __typename?: 'Query';
  coercion: Scalars['Boolean']['output'];
  complexity: Scalars['Boolean']['output'];
  date: Scalars['Boolean']['output'];
  error: Scalars['Boolean']['output'];
  jsonEncoding: Scalars['String']['output'];
  path?: Maybe<Array<Maybe<Element>>>;
  viewer?: Maybe<Viewer>;
};


export type QueryCoercionArgs = {
  value?: InputMaybe<Array<ListCoercion>>;
};


export type QueryComplexityArgs = {
  value: Scalars['Int']['input'];
};


export type QueryDateArgs = {
  filter: DateFilter;
};


export type QueryErrorArgs = {
  type?: InputMaybe<ErrorType>;
};

export type RemoteModelWithOmitempty = {
  __typename?: 'RemoteModelWithOmitempty';
  newDesc?: Maybe<Scalars['String']['output']>;
};

export type User = {
  __typename?: 'User';
  likes: Array<Scalars['String']['output']>;
  name: Scalars['String']['output'];
};

export type Viewer = {
  __typename?: 'Viewer';
  user?: Maybe<User>;
};

export type CoercionQueryVariables = Exact<{
  value?: InputMaybe<Array<ListCoercion> | ListCoercion>;
}>;


export type CoercionQuery = { __typename?: 'Query', coercion: boolean };

export type ComplexityQueryVariables = Exact<{
  value: Scalars['Int']['input'];
}>;


export type ComplexityQuery = { __typename?: 'Query', complexity: boolean };

export type DateQueryVariables = Exact<{
  filter: DateFilter;
}>;


export type DateQuery = { __typename?: 'Query', date: boolean };

export type ErrorQueryVariables = Exact<{
  type?: InputMaybe<ErrorType>;
}>;


export type ErrorQuery = { __typename?: 'Query', error: boolean };

export type JsonEncodingQueryVariables = Exact<{ [key: string]: never; }>;


export type JsonEncodingQuery = { __typename?: 'Query', jsonEncoding: string };

export type PathQueryVariables = Exact<{ [key: string]: never; }>;


export type PathQuery = { __typename?: 'Query', path?: Array<{ __typename?: 'Element', cc: { __typename?: 'Element', error: boolean } } | null> | null };


export const CoercionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"coercion"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"value"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ListCoercion"}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"coercion"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"value"},"value":{"kind":"Variable","name":{"kind":"Name","value":"value"}}}]}]}}]} as unknown as DocumentNode<CoercionQuery, CoercionQueryVariables>;
export const ComplexityDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"complexity"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"value"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"complexity"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"value"},"value":{"kind":"Variable","name":{"kind":"Name","value":"value"}}}]}]}}]} as unknown as DocumentNode<ComplexityQuery, ComplexityQueryVariables>;
export const DateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"date"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"filter"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DateFilter"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"date"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"filter"},"value":{"kind":"Variable","name":{"kind":"Name","value":"filter"}}}]}]}}]} as unknown as DocumentNode<DateQuery, DateQueryVariables>;
export const ErrorDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"error"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"type"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"ErrorType"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"error"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"type"},"value":{"kind":"Variable","name":{"kind":"Name","value":"type"}}}]}]}}]} as unknown as DocumentNode<ErrorQuery, ErrorQueryVariables>;
export const JsonEncodingDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"jsonEncoding"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jsonEncoding"}}]}}]} as unknown as DocumentNode<JsonEncodingQuery, JsonEncodingQueryVariables>;
export const PathDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"path"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"cc"},"name":{"kind":"Name","value":"child"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]}}]} as unknown as DocumentNode<PathQuery, PathQueryVariables>;