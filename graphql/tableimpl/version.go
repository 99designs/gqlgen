package tableimpl

// SupportPackageIsVersion1 is referred to by the code that gqlgen generates in table
// mode. A change to this package that the code generated before it does not work with
// replaces it with SupportPackageIsVersion2, which the generator then refers to, so that
// code generated for another version of the runtime fails to compile on that name and is
// generated again, instead of failing on whatever part of the runtime changed.
//
// No other code should refer to it.
const SupportPackageIsVersion1 = true
