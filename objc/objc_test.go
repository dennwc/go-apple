package objc

import "testing"

func TestGetClass(t *testing.T) {
	c := GetClass("nonExistent")
	if c != nil {
		t.Errorf("expected nil class: %#v", c)
	}
	c = GetClass("Object")
	if c == nil {
		t.Error("failed to get Object class")
	} else if name := c.Name(); name != "Object" {
		t.Errorf("invalid name: %q", name)
	}
}

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
