package registry

// ExitCode reads the key of an exits map: an integer from 0 to 255 written in
// decimal with no sign, no leading zero and no spaces. ok is false for any
// other key.
func ExitCode(key string) (code int, ok bool) {
	if key == "" || len(key) > 3 || (len(key) > 1 && key[0] == '0') {
		return 0, false
	}
	for _, r := range key {
		if r < '0' || r > '9' {
			return 0, false
		}
		code = code*10 + int(r-'0')
	}
	return code, code <= 255
}
