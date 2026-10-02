// Package execbehavior pins down how the generated executor behaves at runtime, so that a
// change to how the executor is generated can be checked against it. It covers arguments
// and input coercion, nullability, resolver errors and panics, directives, @defer,
// subscriptions, complexity and introspection.
//
// Each behavior has its own test file, and most have a schema file of the same name. A
// schema file or a test that uses what another schema file declares names that file in a
// comment, and the tests of complexity and fragments only use the schemas of the other
// behaviors. The tests compare whole responses, data and errors together, so that a
// change in any part of a response shows, and each test or subtest builds the server and
// the state it records into, so that it does not depend on the others.
//
// Some tests pin a known bug: behavior that differs from what the spec or the
// documentation says. Their expected response is the current one, and a comment marks
// them: a line that starts with "KNOWN BUG:" says what happens now, and a line that
// starts with "Expected:" says what should happen and why, and a line that starts with
// "Issue:" links the issue of the bug. A change that fixes such a bug updates the
// expected response, removes the comment and closes the issue.
//
// The subpackages hold the schemas that need a different gqlgen.yml, and
// internal/exectest the helpers that their tests share.
package execbehavior
