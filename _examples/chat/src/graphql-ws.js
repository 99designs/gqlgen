import { createClient } from 'graphql-ws';
import { print } from 'graphql';
import { ApolloLink, Observable } from '@apollo/client';

export class WebSocketLink extends ApolloLink {
  client;

  constructor(options) {
    super();
    this.client = createClient(options);
  }

  request(operation) {
    return new Observable((sink) => {
      return this.client.subscribe(
        { ...operation, query: print(operation.query) },
        {
          next: sink.next.bind(sink),
          complete: sink.complete.bind(sink),
          error: (err) => {
            if (err instanceof Error) {
              return sink.error(err);
            }

            if (err instanceof CloseEvent) {
              return sink.error(
                // reason will be available on clean closes
                new Error(
                  `Socket closed with event ${err.code} ${err.reason || ''}`,
                ),
              );
            }

            return sink.error(
              new Error(
                err
                  .map(({ message }) => message)
                  .join(', '),
              ),
            );
          },
        },
      );
    });
  }
}