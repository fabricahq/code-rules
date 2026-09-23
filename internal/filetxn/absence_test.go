// Exercise advisory absence checks without reading occupied targets or following linked parents.

package filetxn

import (
	"context"
	"os"
	"testing"
)

func TestRequireAbsent(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := RequireAbsent(context.Background(), root, "missing/group/_group.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Stat("missing"); !os.IsNotExist(err) {
		t.Fatal("created missing parent", err)
	}
	if err := root.Mkdir("group", 0700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("group/rule.md", []byte("not valid rule metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RequireAbsent(context.Background(), root, "group/rule.md"); err == nil {
		t.Fatal("accepted existing file")
	}
	if err := RequireAbsent(context.Background(), root, "group"); err == nil {
		t.Fatal("accepted occupied directory")
	}
	if err := root.Symlink("group", "linked"); err != nil {
		t.Fatal(err)
	}
	if err := RequireAbsent(context.Background(), root, "linked/new.md"); err == nil {
		t.Fatal("followed parent symlink")
	}
	if err := RequireAbsent(context.Background(), root, "../outside.md"); err == nil {
		t.Fatal("accepted escaping path")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RequireAbsent(ctx, root, "new.md"); err == nil {
		t.Fatal("ignored cancellation")
	}
}
