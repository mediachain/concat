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
	"SELECT body FROM foo.bar",
	"SELECT id FROM foo.bar",
	"SELECT source FROM foo.bar",
	"SELECT timestamp FROM foo.bar",
	"SELECT (body, source) FROM foo.bar",
	"SELECT (id, namespace, publisher) FROM foo.bar",
	"SELECT COUNT(*) FROM foo.bar",
	"SELECT COUNT(publisher) FROM foo.bar",
	"SELECT namespace FROM *",
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
	"SELECT * FROM foo.bar WHERE publisher = abc AND timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc OR timestamp > 1474000000",
	"SELECT * FROM foo.bar WHERE (publisher = abc AND timestamp > 1474000000) OR timestamp < 1474000000",
	"SELECT * FROM foo.bar WHERE NOT id = abc",
	"SELECT * FROM foo.bar WHERE NOT (id = abc AND publisher = def)",
	"SELECT * FROM foo.bar WHERE publisher = abc AND NOT timestamp < 1474000000",
	"SELECT * FROM foo.bar WHERE publisher = abc AND NOT timestamp < 1474000000 OR timestamp > 1475000000",
	"SELECT * FROM foo.bar WHERE publisher = abc AND NOT (timestamp < 1474000000 OR timestamp > 1475000000)",
	"SELECT * FROM foo.bar WHERE publisher = abc LIMIT 10",
	"SELECT * FROM foo.bar LIMIT 10",
	"SELECT * FROM * WHERE id = abc"}

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

func TestQueryParse(t *testing.T) {
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
		Body:      &pb.Statement_Simple{&pb.SimpleStatement{Object: "QmAAA"}},
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

	_, err = db.Exec("CREATE TABLE Envelope (id VARCHAR(32) PRIMARY KEY, namespace VARCHAR, publisher VARCHAR, source VARCHAR, timestamp INTEGER)")
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
	_, err = db.Exec("INSERT INTO Envelope VALUES (?, ?, ?, ?, ?)", stmt.Id, stmt.Namespace, stmt.Publisher, stmt.Publisher, stmt.Timestamp)

	return err
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
