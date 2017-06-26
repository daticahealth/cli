package git

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %s", err)
	}
	t.Log(wd)
	defer os.Chdir(wd)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to make temp directory: %s", err)
	}

	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Failed to change working directory: %s", err)
	}
	ig := New()
	err = ig.Create()
	if err != nil {
		t.Fatalf("Failed to initialize a git directory: %s", err)
	}

	if !New().Exists() {
		t.Fatalf("Exists returned false for a valid git repo")
	}
}
