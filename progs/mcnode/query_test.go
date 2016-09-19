package main

import (
	"testing"
)

var simpleq []string = []string{
	"SELECT * FROM foo.bar",
	"SELECT body FROM foo.bar",
	"SELECT id FROM foo.bar",
	"SELECT source FROM foo.bar",
	"SELECT timestamp FROM foo.bar",
	"SELECT (body, source) FROM foo.bar",
	"SELECT COUNT(*) FROM foo.bar",
	"SELECT namespace FROM *",
	"SELECT * FROM foo.bar.*",
	"SELECT * FROM foo.bar-baz-with-dashes",
	"SELECT * FROM foo.bar WHERE id = abc",
	"SELECT * FROM foo.bar WHERE publisher = abc",
	"SELECT * FROM foo.bar WHERE source = abc",
	"SELECT * FROM foo.bar WHERE timestamp < 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp <= 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp = 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp >= 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc AND timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc LIMIT 10",
	"SELECT * FROM foo.bar LIMIT 10"}

func TestQuerySyntax(t *testing.T) {
	for _, q := range simpleq {
		p := &QueryParser{Buffer: q}
		p.Init()
		err := p.Parse()
		if err != nil {
			t.Logf("QUERY: %s", q)
			t.Error(err)
		}
	}
}
