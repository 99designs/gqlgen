package config

import "strings"

// commonInitialisms is a set of common initialisms.
// Only add entries that are highly unlikely to be non-initialisms.
// For instance, "ID" is fine (Freudian code is rare), but "AND" is not.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"CSV":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ICMP":  true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"KVK":   true,
	"LHS":   true,
	"PDF":   true,
	"PGP":   true,
	"QPS":   true,
	"QR":    true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"SVG":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"UUID":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}

// GetInitialisms returns the initialisms to capitalize in Go names. If unchanged, default initialisms will be returned
var GetInitialisms = func() map[string]bool {
	return commonInitialisms
}

// GoInitialismsConfig allows to modify the default behavior of naming Go methods, types and properties
type GoInitialismsConfig struct {
	// If true, the Initialisms won't get appended to the default ones but replace them
	ReplaceDefaults bool `yaml:"replace_defaults"`
	// Custom initialisms to be added or to replace the default ones
	Initialisms []string `yaml:"initialisms"`
}

// setInitialisms adjustes GetInitialisms based on its settings.
func (i GoInitialismsConfig) setInitialisms() {
	toUse := i.determineGoInitialisms()
	GetInitialisms = func() map[string]bool {
		return toUse
	}
}

// determineGoInitialisms returns the Go initialims to be used, based on its settings.
func (i GoInitialismsConfig) determineGoInitialisms() (initialismsToUse map[string]bool) {
	if i.ReplaceDefaults {
		initialismsToUse = make(map[string]bool, len(i.Initialisms))
		for _, initialism := range i.Initialisms {
			initialismsToUse[strings.ToUpper(initialism)] = true
		}
	} else {
		initialismsToUse = make(map[string]bool, len(commonInitialisms)+len(i.Initialisms))
		for initialism, value := range commonInitialisms {
			initialismsToUse[strings.ToUpper(initialism)] = value
		}
		for _, initialism := range i.Initialisms {
			initialismsToUse[strings.ToUpper(initialism)] = true
		}
	}
	return initialismsToUse
}
