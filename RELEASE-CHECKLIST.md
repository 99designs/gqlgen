# When gqlgen gets released, the following things need to happen
Assuming the next version is $NEW_VERSION=v0.16.0 or something like that.

1. Update https://github.com/99designs/gqlgen/blob/master/graphql/version.go#L3
2. Update https://github.com/99designs/gqlgen/blob/master/docs/build.sh#L13
4. git commit and push those file changes to master
3. git tag -a $NEW_VERSION -m $NEW_VERSION
4. git push origin $NEW_VERSION
5. git-chglog -o CHANGELOG.md
6. git commit and push the CHANGELOG.md
7. https://github.com/99designs/gqlgen/releases and draft new release, autogenerate the release notes, and Create a discussion for this release
8. Comment on the release discussion with any really important notes (breaking changes)

I used https://github.com/git-chglog/git-chglog to automate the changelog maintenance process for now. We could just as easily use go releaser to make the whole thing automated.

