// Run the check workflow's install step against fake release tools, as GitHub Actions runs it.

package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// fakeReleaseTools are the commands the install step runs. Each logs its arguments. gh downloads a fake archive
// and checksum file, and its attestation check passes only for Code Rules' release workflow on main and
// FAKE_ATTESTATION=pass. sha256sum passes only for FAKE_CHECKSUM=pass. tar extracts a fake executable.
var fakeReleaseTools = map[string]string{
	"gh": `echo "gh $*" >> "$FAKE_LOG"
case "$1 $2" in
"release download") printf 'sums\n' > SHA256SUMS; printf 'archive\n' > code-rules_0.2.0_linux_amd64.tar.gz ;;
"attestation verify")
  case "$*" in *"--repo fabricahq/code-rules --signer-workflow fabricahq/code-rules/.github/workflows/release-planner.yml --source-ref refs/heads/main"*) ;; *) exit 1 ;; esac
  [ "$FAKE_ATTESTATION" = pass ] ;;
*) exit 2 ;;
esac`,
	"sha256sum": `echo "sha256sum $*" >> "$FAKE_LOG"
[ "$*" = "--check --ignore-missing SHA256SUMS" ] && [ "$FAKE_CHECKSUM" = pass ]`,
	"tar": `echo "tar $*" >> "$FAKE_LOG"
printf '#!/bin/sh\n' > code-rules`,
}

// installScript returns the generated workflow's install step script, without its YAML indentation.
func installScript(t *testing.T, workflow string) string {
	t.Helper()
	block := regexp.MustCompile(`(?s)\n        run: \|\n(.*?)\n      - `).FindStringSubmatch(workflow)
	if block == nil {
		t.Fatalf("no install script in:\n%s", workflow)
	}
	return regexp.MustCompile(`(?m)^ {10}`).ReplaceAllString(block[1], "") + "\n"
}

// TestCheckWorkflow_InstallsOnlyVerifiedReleases runs the install step with bash -e, as GitHub Actions does: it adds
// Code Rules to PATH only after the attestation and checksum pass, and stops before extracting when either fails.
func TestCheckWorkflow_InstallsOnlyVerifiedReleases(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal("bash, which GitHub Actions uses, is required to run the install step:", err)
	}
	workflow, err := checkWorkflowFor("0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	script := installScript(t, string(workflow))
	tools := t.TempDir()
	for name, body := range fakeReleaseTools {
		if err := os.WriteFile(filepath.Join(tools, name), []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		name, attestation, checksum string
		installed                   bool
		log                         []string
	}{
		{"verified", "pass", "pass", true, []string{"gh release download v0.2.0", "gh attestation verify SHA256SUMS", "sha256sum --check", "tar -xzf code-rules_0.2.0_linux_amd64.tar.gz code-rules"}},
		{"attestation fails", "fail", "pass", false, []string{"gh release download v0.2.0", "gh attestation verify SHA256SUMS"}},
		{"checksum fails", "pass", "fail", false, []string{"gh release download v0.2.0", "gh attestation verify SHA256SUMS", "sha256sum --check"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := t.TempDir()
			githubPath := filepath.Join(runner, "github-path")
			log := filepath.Join(runner, "log")
			command := exec.Command(bash, "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
			command.Dir = runner
			command.Env = []string{"PATH=" + tools + ":/usr/bin:/bin", "RUNNER_TEMP=" + filepath.Join(runner, "temp"), "GITHUB_PATH=" + githubPath, "GH_TOKEN=token", "FAKE_LOG=" + log, "FAKE_ATTESTATION=" + test.attestation, "FAKE_CHECKSUM=" + test.checksum}
			output, err := command.CombinedOutput()
			if (err == nil) != test.installed {
				t.Fatalf("install step error %v, want success %v:\n%s", err, test.installed, output)
			}
			calls, _ := os.ReadFile(log)
			lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
			if len(lines) != len(test.log) {
				t.Fatalf("ran:\n%s\nwant %d commands starting %q", calls, len(test.log), test.log)
			}
			for i, prefix := range test.log {
				if !strings.HasPrefix(lines[i], prefix) {
					t.Fatalf("command %d is %q, want it to start with %q", i+1, lines[i], prefix)
				}
			}
			path, _ := os.ReadFile(githubPath)
			executable := filepath.Join(runner, "temp", "code-rules", "code-rules")
			_, statErr := os.Stat(executable)
			if test.installed != (string(path) == filepath.Dir(executable)+"\n" && statErr == nil) {
				t.Fatalf("GITHUB_PATH %q and executable %v, want installed %v", path, statErr, test.installed)
			}
		})
	}
}
