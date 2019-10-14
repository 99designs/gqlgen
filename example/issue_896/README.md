This example should build a stable `generated.go` file. If the file content
starts alternating nondeterministically between two outputs, then
[#896](https://github.com/99designs/gqlgen/issues/896) may have regressed.
