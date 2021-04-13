# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added 

* Added a changelog :-) Following the same style as [Apollo Client](https://github.com/apollographql/apollo-client) because that feels like it gives good thanks to the community contributors.
By [@MichaelJCompton](https://github.com/MichaelJCompton) in [#1512](https://github.com/99designs/gqlgen/pull/1512)
* Added support for methods returning `(v, ok)` shaped values to support Prisma Go client. <br />
By [@steebchen](https://github.com/steebchen) in [#1449](https://github.com/99designs/gqlgen/pull/1449)

### Changed

* GraphQL playground updated to 1.7.26.  <br />
By [@ddouglas](https://github.com/ddouglas) in [#1436](https://github.com/99designs/gqlgen/pull/1436)

### Fixed

* v0.12.2 broke the handling of pointers to slices by calling the custom Marshal and Unmarshal functions on the entire slice.  It now correctly calls the custom Marshal and Unmarshal methods for each element in the slice. <br />
By [@ananyasaxena](https://github.com/ananyasaxena) in [#1363](https://github.com/99designs/gqlgen/pull/1363)
* Changes in go1.16 that mean go.mod and go.sum aren't always up to date.  Now `go mod tidy` is run after code generation. <br />
By [@lwc](https://github.com/lwc) in [#1501](https://github.com/99designs/gqlgen/pull/1501)
* Errors in resolving non-nullable arrays were not correctly bubbling up to the next nullable field. <br />
By [@wilhelmeek](https://github.com/wilhelmeek) in [#1480](https://github.com/99designs/gqlgen/pull/1480)
* Fixed a potential deadlock in calling error presenters.  <br />
By [@vektah](https://github.com/vektah) in [#1399](https://github.com/99designs/gqlgen/pull/1399)
* Fixed `collectFields` not correctly respecting alias fields in fragments.  <br />
By [@vmrajas](https://github.com/vmrajas) in [#1341](https://github.com/99designs/gqlgen/pull/1341)


## [0.13.0] - 2020-09-21

Base version at which changelog was introduced.

