# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Fixed


## v0.14.0 - 2021-09-08

### Added

* Added a changelog :-) Following the same style as [Apollo Client](https://github.com/apollographql/apollo-client) because that feels like it gives good thanks to the community contributors. <br />
By [@MichaelJCompton](https://github.com/MichaelJCompton) in [#1512](https://github.com/99designs/gqlgen/pull/1512)
* Added support for methods returning `(v, ok)` shaped values to support Prisma Go client. <br />
By [@steebchen](https://github.com/steebchen) in [#1449](https://github.com/99designs/gqlgen/pull/1449)
* Added a new API to finish an already validated config  <br />
By [@benjaminjkraft](https://github.com/benjaminjkraft) in [#1387](https://github.com/99designs/gqlgen/pull/1387)

### Changed

* Updated to gqlparser to v2.2.0. <br />
By [@lwc](https://github.com/lwc) in [#1514](https://github.com/99designs/gqlgen/pull/1514)
* GraphQL playground updated to 1.7.26.  <br />
By [@ddouglas](https://github.com/ddouglas) in [#1436](https://github.com/99designs/gqlgen/pull/1436)

### Fixed

* Removed a data race by copying when input fields have default values. <br />
By [@skaji](https://github.com/skaji) in [#1456](https://github.com/99designs/gqlgen/pull/1456)
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
* Return introspection document in stable order.  <br />
* By [@nyergler](https://github.com/nyergler) in [#1497](https://github.com/99designs/gqlgen/pull/1497)

## v0.13.0 - 2020-09-21

Base version at which changelog was introduced.

