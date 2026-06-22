# Contribution Guidelines

Want to contribute to gqlgen? Here are some guidelines for how we accept help.

## Getting in Touch

Our [discord](https://discord.gg/DYEq3EMs4U) server is the best place to ask questions or get advice on using gqlgen.

## Reporting Bugs and Issues

 We use [GitHub Issues](https://github.com/99designs/gqlgen/issues) to track bugs, so please do a search before submitting to ensure your problem isn't already tracked.

### New Issues

Please provide the expected and observed behaviours in your issue. A minimal GraphQL schema or configuration file should be provided where appropriate.

## Proposing a Change

If you intend to implement a feature for gqlgen, or make a non-trivial change to the current implementation, we recommend [first filing an issue](https://github.com/99designs/gqlgen/issues/new) marked with the `proposal` tag, so that the engineering team can provide guidance and feedback on the direction of an implementation.  This also help ensure that other people aren't also working on the same thing.

Bug fixes are welcome and should come with appropriate test coverage.

New features should be made against the `next` branch.

### Coding Guidelines

Before opening a pull request, please read [RULES.md](./RULES.md). It describes the coding
practices we expect — error handling, context discipline, concurrency, testing, and how to
work with generated code and the `_examples` module. If you use an AI coding assistant, point
it at that file; PRs are reviewed against those standards regardless of how the code was
produced. Attribute contributions to the people who directed the work, not to the tool — do not add AI-tool `Co-authored-by:` lines or "Generated with …" trailers to commits or PRs.

### License

By contributing to gqlgen, you agree that your contributions will be licensed under its MIT license.
