// Package ghfixture installs a network-free GitHub CLI that returns declared responses and records every call.
package ghfixture

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Response is what the fake returns for one exact argument list.
type Response struct {
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
}

// Call is one recorded invocation, in call order.
type Call struct {
	Args  []string
	Stdin string
}

// Fake owns its state directory; the caller owns the bin directory it was installed in.
type Fake struct {
	state string
}

// Install writes a gh executable into binDir. A call whose arguments exactly match a response prints its Stdout
// and Stderr and exits with its ExitCode; any other call fails with exit status 1. The script uses only absolute
// paths to system tools, so it works when PATH contains nothing but binDir.
func Install(binDir string, responses []Response) (*Fake, error) {
	state, err := os.MkdirTemp(binDir, "gh-state-*")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(state, "count"), []byte("0\n"), 0600); err != nil {
		return nil, err
	}
	for i, response := range responses {
		prefix := filepath.Join(state, "response-"+strconv.Itoa(i))
		files := map[string][]byte{".args": joinArgs(response.Args), ".stdout": []byte(response.Stdout), ".stderr": []byte(response.Stderr), ".code": []byte(strconv.Itoa(response.ExitCode) + "\n")}
		for suffix, data := range files {
			if err := os.WriteFile(prefix+suffix, data, 0600); err != nil {
				return nil, err
			}
		}
	}
	script := `#!/bin/sh
state=` + quote(state) + `
read -r n < "$state/count"
n=$((n + 1))
printf '%s\n' "$n" > "$state/count"
: > "$state/call-$n.args"
for arg in "$@"; do printf '%s\0' "$arg" >> "$state/call-$n.args"; done
/bin/cat > "$state/call-$n.stdin"
i=0
while [ -f "$state/response-$i.args" ]; do
  if /usr/bin/cmp -s "$state/call-$n.args" "$state/response-$i.args"; then
    /bin/cat "$state/response-$i.stdout"
    /bin/cat "$state/response-$i.stderr" >&2
    read -r code < "$state/response-$i.code"
    exit "$code"
  fi
  i=$((i + 1))
done
echo "Unexpected gh invocation" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(script), 0700); err != nil {
		return nil, err
	}
	return &Fake{state: state}, nil
}

// Calls returns every recorded invocation in order, including unexpected ones.
func (f *Fake) Calls() ([]Call, error) {
	data, err := os.ReadFile(filepath.Join(f.state, "count"))
	if err != nil {
		return nil, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("read fake gh call count: %v", err)
	}
	calls := make([]Call, 0, count)
	for n := 1; n <= count; n++ {
		prefix := filepath.Join(f.state, "call-"+strconv.Itoa(n))
		args, err := os.ReadFile(prefix + ".args")
		if err != nil {
			return nil, err
		}
		stdin, err := os.ReadFile(prefix + ".stdin")
		if err != nil {
			return nil, err
		}
		calls = append(calls, Call{Args: splitArgs(args), Stdin: string(stdin)})
	}
	return calls, nil
}

// joinArgs terminates each argument with NUL, matching the script's recording format.
func joinArgs(args []string) []byte {
	var buffer bytes.Buffer
	for _, arg := range args {
		buffer.WriteString(arg)
		buffer.WriteByte(0)
	}
	return buffer.Bytes()
}

// splitArgs reverses joinArgs; an empty recording is a call without arguments.
func splitArgs(data []byte) []string {
	args := []string{}
	for _, part := range bytes.SplitAfter(data, []byte{0}) {
		if len(part) > 0 {
			args = append(args, string(bytes.TrimSuffix(part, []byte{0})))
		}
	}
	return args
}

// quote escapes a trusted local path as one shell word.
func quote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
