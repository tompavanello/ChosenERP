package backup

import "testing"

func TestCollectAttachment(t *testing.T) {
	set := map[string]struct{}{}
	collectAttachment("/api/v1/attachments/abc123.jpg", set)
	collectAttachment("/api/v1/attachments/../../etc/passwd", set)
	collectAttachment("https://exemplo.com/outro.png", set)
	collectAttachment("", set)

	if _, ok := set["abc123.jpg"]; !ok {
		t.Errorf("esperava abc123.jpg no conjunto")
	}
	if _, ok := set["passwd"]; !ok {
		t.Errorf("esperava o basename passwd (path traversal neutralizado)")
	}
	if len(set) != 2 {
		t.Errorf("esperava 2 arquivos, tem %d: %v", len(set), set)
	}
}

func TestWalkForFiles(t *testing.T) {
	payload := map[string]any{
		"tables": map[string]any{
			"members": []any{
				map[string]any{"photo_url": "/api/v1/attachments/foto.png"},
			},
			"study_materials": []any{
				map[string]any{"disk_name": "material.pdf", "kind": "file"},
			},
		},
	}
	set := map[string]struct{}{}
	walkForFiles(payload, set)

	for _, want := range []string{"foto.png", "material.pdf"} {
		if _, ok := set[want]; !ok {
			t.Errorf("esperava %q no conjunto: %v", want, set)
		}
	}
}

func TestSanitizeSlug(t *testing.T) {
	cases := map[string]string{
		"igreja-central": "igreja-central",
		"Igreja Central": "igreja-central",
		"":               "igreja",
		"!!!":            "igreja",
	}
	for in, want := range cases {
		if got := sanitizeSlug(in); got != want {
			t.Errorf("sanitizeSlug(%q) = %q, esperava %q", in, got, want)
		}
	}
}

func TestCountRows(t *testing.T) {
	payload := map[string]any{
		"tables": map[string]any{
			"members": []any{map[string]any{"id": "1"}, map[string]any{"id": "2"}},
		},
		"children": map[string]any{
			"member_relationships": []any{map[string]any{"kind": "spouse"}},
		},
		"users": []any{map[string]any{"id": "u1"}},
	}
	counts := countRows(payload)
	if counts["members"] != 2 {
		t.Errorf("members = %d, esperava 2", counts["members"])
	}
	if counts["_tables"] != 2 {
		t.Errorf("_tables = %d, esperava 2", counts["_tables"])
	}
	if counts["member_relationships"] != 1 {
		t.Errorf("member_relationships = %d, esperava 1", counts["member_relationships"])
	}
	if counts["users"] != 1 {
		t.Errorf("users = %d, esperava 1", counts["users"])
	}
}
