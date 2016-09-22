package query

import (
	pb "github.com/mediachain/concat/proto"
	"reflect"
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
	"SELECT * FROM foo.bar WHERE timestamp != 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp >= 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc AND timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc OR timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE (publisher = abc AND timestamp > 1474000000) OR timestamp < 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc LIMIT 10",
	"SELECT * FROM foo.bar LIMIT 10"}

func checkError(t *testing.T, where string, err error) {
	if err != nil {
		t.Logf("QUERY: %s", where)
		t.Error(err)
	}
}

func checkErrorNow(t *testing.T, where string, err error) {
	if err != nil {
		t.Logf("QUERY: %s", where)
		t.Log(err)
		t.FailNow()
	}
}

func checkResultLen(t *testing.T, where string, res []interface{}, xlen int) bool {
	if len(res) == xlen {
		return true
	}

	t.Logf("QUERY: %s", where)
	t.Errorf("Bad result length: expected %d elements, but got %v", xlen, res)
	return false
}

func checkContains(t *testing.T, where string, res []interface{}, val interface{}) {
	for _, v := range res {
		if reflect.DeepEqual(v, val) {
			return
		}
	}
	t.Logf("QUERY: %s", where)
	t.Errorf("%v is not in result set", val)
}

func TestQuerySyntax(t *testing.T) {
	for _, qs := range simpleq {
		_, err := ParseQuery(qs)
		checkError(t, qs, err)
	}
}

func TestQueryEval(t *testing.T) {
	a := &pb.Statement{
		Id:        "a",
		Publisher: "A",
		Namespace: "foo.a",
		Body:      &pb.Statement_Simple{&pb.SimpleStatement{Object: "QmAAA"}},
		Timestamp: 100}

	b := &pb.Statement{
		Id:        "b",
		Publisher: "B",
		Namespace: "foo.b",
		Body:      &pb.Statement_Simple{&pb.SimpleStatement{Object: "QmBBB"}},
		Timestamp: 200}

	c := &pb.Statement{
		Id:        "c",
		Publisher: "A",
		Namespace: "bar.c",
		Body:      &pb.Statement_Simple{&pb.SimpleStatement{Object: "QmCCC"}},
		Timestamp: 300}

	stmts := []*pb.Statement{a, b, c}

	// check namespace filter
	qs := "SELECT * FROM *"
	q, err := ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err := EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 3) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM foo.*"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

	// check all simple selectors
	qs = "SELECT body FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a.Body)
	}

	qs = "SELECT id FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, "a")
	}

	qs = "SELECT publisher FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, "A")
	}

	qs = "SELECT source FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, "A")
	}

	qs = "SELECT namespace FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, "foo.a")
	}

	qs = "SELECT timestamp FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, int64(100))
	}

	// check compound selection
	qs = "SELECT (id, publisher) FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, map[string]interface{}{"id": "a", "publisher": "A"})
	}

	// check the function
	qs = "SELECT COUNT(*) FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 1)
	}

	// check the limits -- order is unpredictable, so just check the count
	qs = "SELECT COUNT(*) FROM * LIMIT 1"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 1)
	}

	// check simple selection criteria
	qs = "SELECT * FROM * WHERE id = a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE publisher = A"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE source = A"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp < 200"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE timestamp <= 200"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE timestamp = 200"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE timestamp != 200"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp >= 200"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, b)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp > 200"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	// check compound selection criteria
	qs = "SELECT * FROM * WHERE timestamp >= 200 AND publisher = A"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp < 200 OR publisher = B"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE publisher = B OR (publisher = A AND timestamp < 200)"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

}
