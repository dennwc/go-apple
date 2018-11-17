package objc

import "testing"

func TestListClasses(t *testing.T) {
	list := ListClasses()
	if len(list) == 0 {
		t.Error("no classes")
	}
	t.Logf("classes: %v", list)

	left := map[string]struct{}{
		"Object":   {},
		"Protocol": {},
	}
	for _, c := range list {
		name := c.Name()
		delete(left, name)
	}
	if len(left) != 0 {
		t.Errorf("failed to find classes: %v", left)
	}
}
