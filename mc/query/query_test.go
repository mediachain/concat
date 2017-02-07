package query

import (
	"database/sql"
	ggproto "github.com/gogo/protobuf/proto"
	_ "github.com/mattn/go-sqlite3"
	pb "github.com/mediachain/concat/proto"
	"reflect"
	"testing"
)

var simpleq []string = []string{
	"SELECT * FROM foo.bar",
	"SELECT id FROM foo.bar",
	"SELECT body FROM foo.bar",
	"SELECT publisher FROM foo.bar",
	"SELECT source FROM foo.bar",
	"SELECT timestamp FROM foo.bar",
	"SELECT counter FROM foo.bar",
	"SELECT namespace FROM *",
	"SELECT (body, source) FROM foo.bar",
	"SELECT (id, namespace, publisher) FROM foo.bar",
	"SELECT COUNT(*) FROM foo.bar",
	"SELECT COUNT(id) FROM foo.bar",
	"SELECT COUNT(body) FROM foo.bar",
	"SELECT COUNT(publisher) FROM foo.bar",
	"SELECT COUNT(source) FROM foo.bar",
	"SELECT COUNT(timestamp) FROM foo.bar",
	"SELECT COUNT(counter) FROM foo.bar",
	"SELECT COUNT(namespace) FROM *",
	"SELECT MIN(timestamp) FROM foo.bar",
	"SELECT MAX(timestamp) FROM foo.bar",
	"SELECT MIN(counter) FROM foo.bar",
	"SELECT MAX(counter) FROM foo.bar",
	"SELECT (id, namespace, publisher) FROM *",
	"SELECT * FROM foo.bar.*",
	"SELECT * FROM foo.bar-baz-with-dashes",
	"SELECT * FROM foo.bar WHERE id = abc",
	"SELECT * FROM foo.bar WHERE id != abc",
	"SELECT * FROM foo.bar WHERE publisher = abc",
	"SELECT * FROM foo.bar WHERE publisher != abc",
	"SELECT * FROM foo.bar WHERE source = abc",
	"SELECT * FROM foo.bar WHERE source != abc",
	"SELECT * FROM foo.bar WHERE timestamp < 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp <= 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp = 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp != 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp >= 1474000000",
	"SELECT * FROM foo.bar WHERE timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE counter < 10",
	"SELECT * FROM foo.bar WHERE counter <= 10",
	"SELECT * FROM foo.bar WHERE counter = 10",
	"SELECT * FROM foo.bar WHERE counter != 10",
	"SELECT * FROM foo.bar WHERE counter >= 10",
	"SELECT * FROM foo.bar WHERE counter > 10",
	"SELECT * FROM foo.bar WHERE publisher = abc AND timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc OR timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE (publisher = abc AND timestamp > 1474000000) OR timestamp < 1474000000",
	"SELECT * FROM foo.bar WHERE (publisher = abc AND timestamp > 1474000000) OR counter > 10",
	"SELECT * FROM foo.bar WHERE NOT id = abc",
	"SELECT * FROM foo.bar WHERE NOT (id = abc AND publisher = def)",
	"SELECT * FROM foo.bar WHERE publisher = abc AND NOT timestamp < 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc AND NOT timestamp < 1474000000 OR counter > 10",
	"SELECT * FROM foo.bar WHERE publisher = abc AND NOT (timestamp < 1474000000 OR counter > 10)",
	"SELECT * FROM foo.bar WHERE publisher = abc LIMIT 10",
	"SELECT * FROM foo.bar WHERE wki = mywki:abc",
	"SELECT * FROM foo.bar WHERE wki = mywki:abc-defg_123-ABC/xyz.XYZ",
	"SELECT * FROM foo.bar LIMIT 10",
	"SELECT * FROM * WHERE id = abc",
	"SELECT * FROM * ORDER BY id",
	"SELECT * FROM * ORDER BY namespace",
	"SELECT * FROM * ORDER BY publisher",
	"SELECT * FROM * ORDER BY source",
	"SELECT * FROM * ORDER BY timestamp",
	"SELECT * FROM * ORDER BY counter",
	"SELECT * FROM * ORDER BY counter ASC",
	"SELECT * FROM * ORDER BY counter DESC",
	"SELECT * FROM * ORDER BY namespace, counter",
	"SELECT * FROM * ORDER BY namespace ASC, counter",
	"SELECT * FROM * ORDER BY namespace DESC, counter",
	"SELECT * FROM * ORDER BY namespace DESC, counter ASC",
	"SELECT * FROM * WHERE timestamp > 1474000000 ORDER BY counter",
	"SELECT * FROM * ORDER BY counter LIMIT 10",
	"SELECT * FROM * WHERE timestamp > 1474000000 ORDER BY counter LIMIT 10",
}

var delq []string = []string{
	"DELETE FROM *",
	"DELETE FROM foo.bar",
	"DELETE FROM foo.*",
	"DELETE FROM foo.bar WHERE id = abc",
	"DELETE FROM foo.bar WHERE id != abc",
	"DELETE FROM foo.bar WHERE publisher = abc",
	"DELETE FROM foo.bar WHERE publisher != abc",
	"DELETE FROM foo.bar WHERE source = abc",
	"DELETE FROM foo.bar WHERE source != abc",
	"DELETE FROM foo.bar WHERE timestamp < 1474000000",
	"DELETE FROM foo.bar WHERE timestamp <= 1474000000",
	"DELETE FROM foo.bar WHERE timestamp = 1474000000",
	"DELETE FROM foo.bar WHERE timestamp != 1474000000",
	"DELETE FROM foo.bar WHERE timestamp >= 1474000000",
	"DELETE FROM foo.bar WHERE timestamp > 1474000000",
	"DELETE FROM foo.bar WHERE counter < 10",
	"DELETE FROM foo.bar WHERE counter <= 10",
	"DELETE FROM foo.bar WHERE counter = 10",
	"DELETE FROM foo.bar WHERE counter != 10",
	"DELETE FROM foo.bar WHERE counter >= 10",
	"DELETE FROM foo.bar WHERE counter > 10",
	"DELETE FROM foo.bar WHERE publisher = abc AND timestamp > 1474000000",
	"DELETE FROM foo.bar WHERE publisher = abc OR timestamp > 1474000000",
	"DELETE FROM foo.bar WHERE (publisher = abc AND timestamp > 1474000000) OR timestamp < 1474000000",
	"DELETE FROM foo.bar WHERE (publisher = abc AND timestamp > 1474000000) OR counter > 10",
	"DELETE FROM foo.bar WHERE NOT id = abc",
	"DELETE FROM foo.bar WHERE NOT (id = abc AND publisher = def)",
	"DELETE FROM foo.bar WHERE publisher = abc AND NOT timestamp < 1474000000",
	"DELETE FROM foo.bar WHERE publisher = abc AND NOT timestamp < 1474000000 OR counter > 10",
	"DELETE FROM foo.bar WHERE publisher = abc AND NOT (timestamp < 1474000000 OR counter > 10)",
	"DELETE FROM * WHERE id = abc",
	"DELETE FROM * LIMIT 10",
	"DELETE FROM * WHERE id = abc LIMIT 10",
}

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

func checkBool(t *testing.T, where string, e bool) {
	if !e {
		t.Logf("QUERY: %s", where)
		t.Errorf("boolean condition failed")
	}
}

func TestQueryParse(t *testing.T) {
	for _, qs := range simpleq {
		q, err := ParseQuery(qs)
		checkError(t, qs, err)
		checkBool(t, qs, q.Op == OpSelect)
	}
}

func TestQueryParseDelete(t *testing.T) {
	for _, qs := range delq {
		q, err := ParseQuery(qs)
		checkError(t, qs, err)
		checkBool(t, qs, q.Op == OpDelete)
	}
}

func TestQueryEval(t *testing.T) {
	a := &pb.Statement{
		Id:        "a",
		Publisher: "A",
		Namespace: "foo.a",
		Body:      &pb.StatementBody{&pb.StatementBody_Simple{&pb.SimpleStatement{Object: "QmAAA", Refs: []string{"aaa"}}}},
		Timestamp: 100}

	b := &pb.Statement{
		Id:        "b",
		Publisher: "B",
		Namespace: "foo.b",
		Body:      &pb.StatementBody{&pb.StatementBody_Simple{&pb.SimpleStatement{Object: "QmBBB", Refs: []string{"bbb"}}}},
		Timestamp: 200}

	c := &pb.Statement{
		Id:        "c",
		Publisher: "A",
		Namespace: "bar.c",
		Body:      &pb.StatementBody{&pb.StatementBody_Simple{&pb.SimpleStatement{Object: "QmCCC", Refs: []string{"ccc"}}}},
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

	// check the functions
	qs = "SELECT COUNT(*) FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 1)
	}

	qs = "SELECT MIN(timestamp) FROM *"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, int64(100))
	}

	qs = "SELECT MAX(timestamp) FROM *"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, int64(300))
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

	qs = "SELECT * FROM * WHERE publisher != A"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
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

	qs = "SELECT * FROM * WHERE timestamp > 100 AND NOT publisher = A"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE NOT (timestamp < 200 OR publisher = B)"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	// check compound selection
	qs = "SELECT (id, publisher, timestamp) FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, map[string]interface{}{"id": "a", "publisher": "A", "timestamp": int64(100)})
	}

	qs = "SELECT (id, publisher, body) FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, map[string]interface{}{"id": "a", "publisher": "A", "body": a.Body})
	}

	qs = "SELECT (*, id, publisher, body) FROM foo.a"
	q, err = ParseQuery(qs)
	checkErrorNow(t, qs, err)

	res, err = EvalQuery(q, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, map[string]interface{}{"id": "a", "publisher": "A", "*": a, "body": a.Body})
	}

	// check wki selection
	qs = "SELECT * FROM * WHERE wki = aaa"
	res, err = parseEval(qs, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE wki = bbb OR wki = ccc"
	res, err = parseEval(qs, stmts)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, b)
		checkContains(t, qs, res, c)
	}

}

func parseEval(qs string, stmts []*pb.Statement) ([]interface{}, error) {
	q, err := ParseQuery(qs)
	if err != nil {
		return nil, err
	}

	res, err := EvalQuery(q, stmts)
	return res, err
}

func TestQueryCompile(t *testing.T) {
	for _, qs := range simpleq {
		q, err := ParseQuery(qs)
		checkErrorNow(t, qs, err)
		_, _, err = CompileQuery(q)
		checkErrorNow(t, qs, err)
		//fmt.Printf("Compile %s -> %s\n", qs, sqlq)
	}
}

func TestPBWTF(t *testing.T) {
	a := &pb.Statement{
		Id:        "a",
		Publisher: "A",
		Namespace: "foo.a",
		Body:      &pb.StatementBody{&pb.StatementBody_Simple{&pb.SimpleStatement{Object: "QmAAA"}}},
		Timestamp: 100}

	bytes, err := ggproto.Marshal(a)
	checkErrorNow(t, "Marshal", err)

	aa := new(pb.Statement)
	err = ggproto.Unmarshal(bytes, aa)
	checkErrorNow(t, "Unmarshal", err)

	if !reflect.DeepEqual(a, aa) {
		t.Log("Marshal/Unmarshal is not idempotent!!!")
		t.Logf("Expected: %v; Got: %v", a, aa)
		t.FailNow()
	}
}

func TestQueryCompileEval(t *testing.T) {
	a := &pb.Statement{
		Id:        "a",
		Publisher: "A",
		Namespace: "foo.a",
		Body:      &pb.StatementBody{&pb.StatementBody_Simple{&pb.SimpleStatement{Object: "QmAAA", Refs: []string{"aaa"}}}},
		Timestamp: 100}

	b := &pb.Statement{
		Id:        "b",
		Publisher: "B",
		Namespace: "foo.b",
		Body:      &pb.StatementBody{&pb.StatementBody_Simple{&pb.SimpleStatement{Object: "QmBBB", Refs: []string{"bbb"}}}},
		Timestamp: 200}

	c := &pb.Statement{
		Id:        "c",
		Publisher: "A",
		Namespace: "bar.c",
		Body:      &pb.StatementBody{&pb.StatementBody_Simple{&pb.SimpleStatement{Object: "QmCCC", Refs: []string{"ccc"}}}},
		Timestamp: 300}

	stmts := []*pb.Statement{a, b, c}

	db, err := makeStmtDb()
	checkErrorNow(t, "makeStmtDb", err)

	for _, stmt := range stmts {
		err = insertStmt(db, stmt)
		checkErrorNow(t, "insertStmt", err)
	}

	// basic statement queries
	qs := "SELECT * FROM *"
	res, err := parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 3) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM foo.a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE id = a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM foo.a WHERE id = a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM foo.* WHERE id = a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM foo.* WHERE id = c"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	checkResultLen(t, qs, res, 0)

	// basic counting
	qs = "SELECT COUNT(*) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 3)
	}

	qs = "SELECT COUNT(id) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 3)
	}

	qs = "SELECT COUNT(namespace) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 3)
	}

	qs = "SELECT COUNT(publisher) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 2)
	}

	qs = "SELECT COUNT(id) FROM foo.a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 1)
	}

	qs = "SELECT COUNT(namespace) FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, 2)
	}

	// min/max timestamps
	qs = "SELECT MIN(timestamp) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, int64(100))
	}

	qs = "SELECT MAX(timestamp) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, int64(300))
	}

	// min/max counters
	qs = "SELECT MIN(counter) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, int64(1))
	}

	qs = "SELECT MAX(counter) FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, int64(3))
	}

	// all simple selectors
	qs = "SELECT body FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a.Body)
		checkContains(t, qs, res, b.Body)
	}

	qs = "SELECT body FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 3) {
		checkContains(t, qs, res, a.Body)
		checkContains(t, qs, res, b.Body)
		checkContains(t, qs, res, c.Body)
	}

	qs = "SELECT id FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, "a")
		checkContains(t, qs, res, "b")
	}

	qs = "SELECT id FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 3) {
		checkContains(t, qs, res, "a")
		checkContains(t, qs, res, "b")
		checkContains(t, qs, res, "c")
	}

	qs = "SELECT publisher FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, "A")
		checkContains(t, qs, res, "B")
	}

	qs = "SELECT publisher FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, "A")
		checkContains(t, qs, res, "B")
	}

	qs = "SELECT source FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, "A")
		checkContains(t, qs, res, "B")
	}

	qs = "SELECT source FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, "A")
		checkContains(t, qs, res, "B")
	}

	qs = "SELECT timestamp FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, int64(100))
		checkContains(t, qs, res, int64(200))
	}

	qs = "SELECT timestamp FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 3) {
		checkContains(t, qs, res, int64(100))
		checkContains(t, qs, res, int64(200))
		checkContains(t, qs, res, int64(300))
	}

	qs = "SELECT counter FROM foo.*"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, int64(1))
		checkContains(t, qs, res, int64(2))
	}

	qs = "SELECT counter FROM *"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 3) {
		checkContains(t, qs, res, int64(1))
		checkContains(t, qs, res, int64(2))
		checkContains(t, qs, res, int64(3))
	}

	// check compound selection
	qs = "SELECT (id, publisher, timestamp) FROM foo.a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, map[string]interface{}{"id": "a", "publisher": "A", "timestamp": int64(100)})
	}

	qs = "SELECT (id, publisher, body) FROM foo.a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, map[string]interface{}{"id": "a", "publisher": "A", "body": a.Body})
	}

	qs = "SELECT (*, id, publisher, body) FROM foo.a"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, map[string]interface{}{"id": "a", "publisher": "A", "*": a, "body": a.Body})
	}

	// check criteria
	qs = "SELECT * FROM * WHERE publisher = A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM foo.* WHERE publisher = A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE publisher != A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM foo.* WHERE publisher != A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE source = A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM foo.* WHERE source = A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE source != A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM foo.* WHERE source != A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE counter = 1"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE counter = 2"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE counter = 3"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE counter > 1"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, b)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp < 200"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE timestamp <= 200"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE timestamp = 200"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE timestamp != 200"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp >= 200"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, b)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp > 200"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp >= 200 AND publisher = A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE timestamp < 200 OR publisher = B"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE publisher = B OR (publisher = A AND timestamp < 200)"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE timestamp > 100 AND NOT publisher = A"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, b)
	}

	qs = "SELECT * FROM * WHERE NOT (timestamp < 200 OR publisher = B)"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	// check order by
	qs = "SELECT * FROM * ORDER BY counter LIMIT 1"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * ORDER BY counter DESC LIMIT 1"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * ORDER BY timestamp LIMIT 1"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * ORDER BY timestamp DESC LIMIT 1"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, c)
	}

	// check limit
	qs = "SELECT * FROM * LIMIT 1"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	// check wki
	qs = "SELECT * FROM * WHERE wki = aaa"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 1) {
		checkContains(t, qs, res, a)
	}

	qs = "SELECT * FROM * WHERE wki = bbb OR wki = ccc"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, b)
		checkContains(t, qs, res, c)
	}

	qs = "SELECT * FROM * WHERE wki = aaa OR timestamp > 200"
	res, err = parseCompileEval(db, qs)
	checkErrorNow(t, qs, err)

	if checkResultLen(t, qs, res, 2) {
		checkContains(t, qs, res, a)
		checkContains(t, qs, res, c)
	}
}

func makeStmtDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE Statement (id VARCHAR(32) PRIMARY KEY, data VARBINARY)")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE Envelope (counter INTEGER PRIMARY KEY AUTOINCREMENT, id VARCHAR(32), namespace VARCHAR, publisher VARCHAR, source VARCHAR, timestamp INTEGER)")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE Refs (id VARCHAR(32), wki VARCHAR)")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func insertStmt(db *sql.DB, stmt *pb.Statement) error {
	bytes, err := ggproto.Marshal(stmt)
	if err != nil {
		return err
	}

	// a real index should use a transaction for both inserts...
	_, err = db.Exec("INSERT INTO Statement VALUES (?, ?)", stmt.Id, bytes)
	if err != nil {
		return err
	}

	// source = publisher only for simple statements
	_, err = db.Exec("INSERT INTO Envelope VALUES (NULL,?, ?, ?, ?, ?)", stmt.Id, stmt.Namespace, stmt.Publisher, stmt.Publisher, stmt.Timestamp)

	for wki, _ := range StatementRefs(stmt) {
		_, err = db.Exec("INSERT INTO Refs VALUES (?, ?)", stmt.Id, wki)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseCompileEval(db *sql.DB, qs string) ([]interface{}, error) {
	q, err := ParseQuery(qs)
	if err != nil {
		return nil, err
	}

	sqlq, rsel, err := CompileQuery(q)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(sqlq)
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0)

	defer rows.Close()
	for rows.Next() {
		obj, err := rsel.Scan(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, obj)
	}

	return res, nil
}
