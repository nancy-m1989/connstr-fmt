package connstr

// maskPlaceholder replaces a secret value's text. It is fixed rather than
// e.g. repeated to the value's length, so Mask's output doesn't itself leak
// how long a password is.
const maskPlaceholder = "****"

// secretKeys lists, by normalized key, the settings whose values are
// credentials rather than configuration, and so should never be printed
// as-is in a log line or error message. Keyed on normalizeKey rather than
// canonicalKey: none of these currently belong to an alias group in
// aliases.go, but a plain map lookup here doesn't depend on that staying
// true.
var secretKeys = map[string]bool{
	"password":      true,
	"pwd":           true,
	"secret":        true,
	"client secret": true,
	"access token":  true,
	"api key":       true,
}

func isSecretKey(key string) bool {
	return secretKeys[normalizeKey(key)]
}

// Mask renders the connection string the same way Format does, except the
// value of every recognized secret key (Password, Pwd, and similar) is
// replaced with a fixed placeholder. Use it anywhere a connection string
// might end up in a log or an error message.
func (cs *ConnectionString) Mask() string {
	return cs.render(func(p Pair) string {
		if isSecretKey(p.Key) {
			return maskPlaceholder
		}
		return formatValue(p.Value)
	})
}
