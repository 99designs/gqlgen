# 📝 mini-habr-API

This project demonstrates how to implement GraphQL subscriptions and cursor-based pagination using gqlgen in a mini API similar to Habr. The implementation follows the official GraphQL documentation specifications and showcases real-time data exchange and efficient data fetching patterns.

## Key GraphQL Features Demonstrated

### GraphQL Subscriptions
The project provides a complete implementation of GraphQL subscriptions using WebSockets, allowing clients to receive real-time updates when new comments are added to posts. This follows the GraphQL subscription specification and shows how to:
- Set up subscription resolvers in gqlgen
- Manage WebSocket connections efficiently
- Implement the publish-subscribe pattern for real-time updates
- Handle connection lifecycle and cleanup

### Cursor-based Pagination
Following GraphQL's Relay Cursor Connections Specification, this project implements efficient cursor-based pagination for comments on posts. This approach:
- Provides stable pagination that works reliably with changing datasets
- Enables clients to navigate large result sets efficiently
- Implements proper pageInfo with hasNextPage and cursor management
- Demonstrates how to structure connection types in gqlgen schemas

This project implements an API for Ozon similar to Habr. It allows working with posts and comments using GraphQL. The system supports creating posts, adding comments, managing comment enabling/disabling for posts, as well as subscribing to new comment notifications.

## 📡 Subscription System  (WebSockets)[1](./graph/subscription.go)[2](./graph/schema.resolvers.go#L317):
- **Publish-Subscribe Pattern**: Implementation of Pub/Sub for real-time notifications about new comments, where components interact through a central channel mechanism
- **Thread-safe subscription management**: Using mutexes for safe access to the subscriber list in a concurrent environment
- **Automatic resource cleanup**: Proper closing of channels and removal of inactive subscribers to prevent memory leaks
- **Asynchrony**: Using non-blocking Go channels for data transmission
- **Error resistance**: Protection against panics when sending data to closed channels using deferred functions
- **Scalability**: Ability to subscribe to events by specific post identifier, providing targeted notification delivery

> **Note on patterns**: Unlike the classic Observer pattern, where observers directly register with the observed object, this project implements the Publish-Subscribe pattern, which introduces an intermediate layer (message broker) between publishers and subscribers. This provides a higher degree of decomposition: publishers don't know about specific subscribers, and subscribers don't know about publishers. Subscriptions are grouped by post identifier, which allows implementing event filtering at the broker level.


## 🚀 Project Launch

To launch the project, follow these steps:

### Prerequisites

- Docker and Docker Compose installed on your system
- Git for cloning the repository

### Installation Steps

1. **Clone the repository**
   ```bash
   git clone https://github.com/nabishec/ozon_habr_api.git
   cd ozon_habr_api
   ```

2. **Launch in Docker containers**
   ```bash
   docker-compose up
   ```

3. **Using the API**
   
   After launching, open your browser and go to:
   ```
   http://localhost:8080
   ```

### Choosing Data Storage

By default, the project uses PostgreSQL for data storage. If you want to use in-memory storage for testing, change the line in the Dockerfile:
```
RUN go build -o main ./cmd/main.go
```
and add the flag [`-s m`](Dockerfile#L7):
```
RUN go build -o main ./cmd/main.go -s m
```

## 📖 API Documentation

Interactive GraphQL playground console is available at:
* **http://localhost:8080**

### GraphQL Query Examples

<details>
    <summary><b>Getting a list of all posts</b></summary>
    
    query{
        posts{
            id
            title 
            text
            authorID
            commentsEnabled
            createDate
        }
    }

</details>

<details>
    <summary><b>Getting a specific post with comments</b></summary>
    
    query {
        post(postID: 1) {
            id
            title
            text
            comments(first: 5) {
                edges {
                    node {
                        id
                        text
                        authorID
                        createDate
                    }
                    cursor
                }
                pageInfo {
                    hasNextPage
                    endCursor
                }
            }
        }
    }

</details>

<details>
    <summary><b>Creating a new post</b></summary>
    
    mutation {
        addPost(postInput: {
            authorID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
            title: "New post"
            text: "Post content"
            commentsEnabled: true
        }) {
            id
            title
            createDate
        }
    }

</details>

<details>
    <summary><b>Creating a new comment</b></summary>
    
    mutation {
        addComment(commentInput: {
            authorID: "123e4567-e89b-12d3-a456-426614174000",
            postID: 1,
            parentID: 1, # ID of existing comment
            text: "This is a reply to comment 1"
        }) {
            id
            text
            parentID
        }   
    }

</details>


<details>
    <summary><b>Subscribing to new comments</b></summary>
    
    subscription {
        commentAdded(postID: 1) {
            id
            text
            authorID
            createDate
        }
    }

</details>

For testing the API, you can use any GraphQL clients, such as Insomnia, Postman, or GraphiQL.

## 📁 Project Structure
<details>
    <summary style="display: inline-flex; align-items: center;">
        <b>Show structure </b>
    </summary>

`ozon_habr_api/`<br>
`├── cmd/`<br>
`│   ├── db_connection/`<br>
`│   │   ├──` [`cache.go`](./cmd/db_connection/cache.go)                (Redis connection and configuration for caching)<br>
`│   │   └──` [`database.go`](./cmd/db_connection/database.go)              (PostgreSQL connection and configuration)<br>
`│   ├── server/`<br>
`│   │   └──` [`server.go`](./cmd/server/server.go)               (GraphQL server setup and launch)<br>
`│   └──` [`main.go`](./cmd/main.go)                     (Main entry point, application setup and launch)<br>
`├── graph/`<br>
`│   ├── model/`<br>
`│   │   └──` [`models_gen.go`](./graph/model/models_gen.go)           (Automatically generated GraphQL models)<br>
`│   ├──` [`generated.go`](./graph/generated.go)                 (Generated GraphQL code (gqlgen))<br>
`│   ├──` [`resolver.go`](./graph/resolver.go)                 (Main GraphQL resolvers)<br>
`│   ├──` [`schema.graphqls`](./graph/schema.graphqls)             (GraphQL schema definition)<br>
`│   ├──` [`schema.resolvers.go`](./graph/schema.resolvers.go)         (GraphQL resolvers implementation)<br>
`│   └──` [`subscription.go`](./graph/subscription.go)         (Implementation of structures and methods for subscription management)<br>
`├── internal/`<br>
`│   ├── handlers/`<br>
`│   │   ├── comment_mutation/`                (Comment mutations logic handlers)<br>
`│   │   │   ├──` [`interface.go`](./internal/handlers/comment_mutation/interface.go)        (Interface for comment mutations)<br>
`│   │   │   └──` [`mutations.go`](./internal/handlers/comment_mutation/mutations.go)        (Comment mutations implementation)<br>
`│   │   ├── comment_query/`                (Comment queries logic handlers)<br>
`│   │   │   ├──` [`interface.go`](./internal/handlers/comment_query/interface.go)        (Interface for comment queries)<br>
`│   │   │   └──` [`query.go`](./internal/handlers/comment_query/query.go)        (Comment queries implementation)<br>
`│   │   ├── post_mutation/`          (Post mutations logic handlers)<br>
`│   │   │   ├──` [`interface.go`](./internal/handlers/post_mutation/interface.go)        (Interface for post mutations)<br>
`│   │   │   └──` [`mutations.go`](./internal/handlers/post_mutation/mutations.go)        (Post mutations implementation)<br>
`│   │   └── post_query/`          (Post queries logic handlers)<br>
`│   │       ├──` [`interface.go`](./internal/handlers/post_query/interface.go)        (Interface for post queries)<br>
`│   │       └──` [`query.go`](./internal/handlers/post_query/query.go)        (Post queries implementation)<br>
`│   ├── pkg/`<br>
`│   │   ├── cursor/`<br>
`│   │   |   └──` [`cursor.go`](./internal/pkg/cursor/cursor.go)        (Functions for working with cursors in pagination)<br>
`│   │   └── errs/`<br>
`│   │       └──` [`errors.go`](./internal/pkg/errs/errors.go)        (Stores business logic errors)<br>
`│   ├── model/`<br>
`│   │   └──` [`model.go`](./internal/model/model.go)                (Internal data models)<br>
`│   └── storage/`<br>
`│       ├── db/` (Implementation of database storage) <br>
`│       │   └──` [`resolvers.go`](./internal/storage/db/resolvers.go)        (Implementation of methods for working with PostgreSQL database)<br>
`│       ├── in-memory/` (Implementation of in-memory data storage) <br>
`│       │   └──` [`resolvers.go`](./internal/storage/in-memory/resolvers.go)        (Implementation of methods for working with in-memory data)<br>
`│       └──` [`interface.go`](./internal/storage/interface.go)            (Interface for data storage (PostgreSQL, in-memory))<br>
`├── migrations/`<br>
`│   └──` [`001_create_tables.up.sql`](./migrations/001_create_tables.up.sql)    (SQL script for database migration (creating tables))<br>
`├──` [`.env`](./.env)                            (Environment variables file (database settings, Redis, etc.))<br>
`├──` [`.gitignore`](./.gitignore)                      (List of ignored files and directories for Git)<br>
`├──` [`docker-compose.yml`](./docker-compose.yml)              (Docker Compose configuration for launching the application and dependencies)<br>
`├──` [`Dockerfile`](./Dockerfile)                      (Instructions for building Docker image)<br>
`├──` [`go.mod`](./go.mod)                          (Go dependencies file)<br>
`├──` [`go.sum`](./go.sum)                          (Go dependencies checksums file)<br>
`├──` [`gqlgen.yml`](./gqlgen.yml)                      (Configuration file for gqlgen)<br>
`├──` [`LICENSE`](./LICENSE)                         (Project license)<br>
`└──` [`README.md`](./README.md)                       (Project description file)<br>

</details>


## 🔧 Stack:
  * Go 1.24
  * GraphQL (gqlgen)
  * PostgreSQL 17
  * Redis 9
  * Docker & Docker Compose
