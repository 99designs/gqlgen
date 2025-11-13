---
title: "Embedding Interfaces"
description: Embed GraphQL interfaces as Go structs using @goEmbedInterface directive
linkTitle: embedded_interfaces
menu: { main: { parent: "recipes" } }
---

Embedding a GraphQL interface in a Go struct lets you share behavior without duplicating fields. Mark the interface with
`@goEmbedInterface` and gqlgen would generate `Base{Interface}` struct and embed it to all implementors instead of copying fields.

### Core Features

- **Create base struct**: Interfaces marked with `@goEmbedInterface` get `Base{Interface}` structs generated for embedding.
- **Resolve chained embedding**: For annotated interfaces, modelgen walks interface implementation hierarchy and reuses already embedded types, so implementing `A & B & C` keeps a single copy of shared parents. When some intermediate interfaces are not annotated, their fields are included directly in the next implementor struct.

### Example

```graphql
directive @goEmbedInterface on INTERFACE

type Query {
  product(id: ID!): Product!
}

interface Node @goEmbedInterface {
  id: ID!
}

interface Ownable @goEmbedInterface {
  owner: String!
}

type Product implements Node & Ownable {
  id: ID!
  owner: String!
  name: String!
}
```

Configuration:

```yaml
# gqlgen.yml
schema:
  - schema.graphqls
model:
  filename: graph/model/models_gen.go
```

gqlgen generates `BaseNode` and `BaseOwnable` structs (only for interfaces with `@goEmbedInterface`), and `Product` embeds both:

```startLine:endLine:graph/model/models_gen.go
type BaseNode struct {
	ID string `json:"id"`
}

func (BaseNode) IsNode() {}

type BaseOwnable struct {
	Owner string `json:"owner"`
}

func (BaseOwnable) IsOwnable() {}

type Product struct {
	BaseNode
	BaseOwnable
	Name string `json:"name"`
}

func (Product) IsNode() {}
func (Product) IsOwnable() {}
```

In resolvers, call methods that return base implementations—no need to construct interface fields manually:

```startLine:endLine:graph/resolver.go
func (r *queryResolver) Product(ctx context.Context, id string) (*model.Product, error) {
	node, err := r.productService.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}

	owner, err := r.productService.GetOwner(ctx, id)
	if err != nil {
		return nil, err
	}

	name, err := r.productService.GetName(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.Product{
		BaseNode:    node,  // embeds ID from service
		BaseOwnable: owner, // embeds Owner from service
		Name:        name,
	}, nil
}
```

**Limitations**

- **Diamond problem**: When an interface implements multiple parent interfaces that both have `@goEmbedInterface`, both base structs are embedded. This works correctly as long as there are no field name conflicts, which prevents embedding of both parents.
- **Covariant overrides**: When a type implements an interface but uses a more specific type for a field (e.g., interface declares `data: NodeData!` but implementation uses `data: ProductNodeData!`), gqlgen skips embedding the base struct for that interface and generates explicit fields instead.
