package main

import (
	"context"
	"database/sql"
	ggproto "github.com/gogo/protobuf/proto"
	_ "github.com/mattn/go-sqlite3"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"log"
	"os"
	"path"
)

type SQLDB struct {
	db                 *sql.DB
	insertStmtData     *sql.Stmt
	insertStmtEnvelope *sql.Stmt
	selectStmtData     *sql.Stmt
}

func (sdb *SQLDB) Put(stmt *pb.Statement) error {
	bytes, err := ggproto.Marshal(stmt)
	if err != nil {
		return err
	}

	tx, err := sdb.db.Begin()
	if err != nil {
		return err
	}

	xstmt := tx.Stmt(sdb.insertStmtData)
	_, err = xstmt.Exec(stmt.Id, bytes)
	if err != nil {
		tx.Rollback()
		return err
	}

	xstmt = tx.Stmt(sdb.insertStmtEnvelope)
	// XXX source = publisher only for simple statements
	_, err = xstmt.Exec(stmt.Id, stmt.Namespace, stmt.Publisher, stmt.Publisher, stmt.Timestamp)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (sdb *SQLDB) Get(id string) (*pb.Statement, error) {
	row := sdb.selectStmtData.QueryRow(id)

	var bytes []byte
	err := row.Scan(&bytes)
	if err != nil {
		if err == sql.ErrNoRows {
			err = UnknownStatement
		}

		return nil, err
	}

	stmt := new(pb.Statement)
	err = ggproto.Unmarshal(bytes, stmt)
	if err != nil {
		return nil, err
	}

	return stmt, nil
}

func (sdb *SQLDB) Query(q *mcq.Query) ([]interface{}, error) {
	sq, rsel, err := mcq.CompileQuery(q)
	if err != nil {
		return nil, err
	}

	rows, err := sdb.db.Query(sq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]interface{}, 0)
	for rows.Next() {
		obj, err := rsel.Scan(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, obj)
	}

	return res, nil
}

func (sdb *SQLDB) QueryStream(ctx context.Context, q *mcq.Query) (<-chan interface{}, error) {
	sq, rsel, err := mcq.CompileQuery(q)
	if err != nil {
		return nil, err
	}

	rows, err := sdb.db.Query(sq)
	if err != nil {
		return nil, err
	}

	ch := make(chan interface{})
	go func() {
		defer close(ch)
		defer rows.Close()

		for rows.Next() {
			obj, err := rsel.Scan(rows)
			if err != nil {
				log.Printf("Error retrieving query result: %s", err.Error())
				return
			}

			select {
			case ch <- obj:
				continue
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (sdb *SQLDB) Close() error {
	return sdb.db.Close()
}

func (sdb *SQLDB) createTables() error {
	_, err := sdb.db.Exec("CREATE TABLE Statement (id VARCHAR(128) PRIMARY KEY, data VARBINARY)")
	if err != nil {
		return err
	}

	_, err = sdb.db.Exec("CREATE TABLE Envelope (id VARCHAR(128) PRIMARY KEY, namespace VARCHAR, publisher VARCHAR, source VARCHAR, timestamp INTEGER)")
	return err
}

func (sdb *SQLDB) prepareStatements() error {
	stmt, err := sdb.db.Prepare("INSERT INTO Statement VALUES (?, ?)")
	if err != nil {
		return err
	}
	sdb.insertStmtData = stmt

	stmt, err = sdb.db.Prepare("INSERT INTO Envelope VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	sdb.insertStmtEnvelope = stmt

	stmt, err = sdb.db.Prepare("SELECT data FROM Statement WHERE id = ?")
	if err != nil {
		return err
	}
	sdb.selectStmtData = stmt

	return nil
}

// SQLite backend
type SQLiteDB struct {
	SQLDB
}

func (sdb *SQLiteDB) Open(home string) error {
	var dbpath string
	var mktables bool

	if home == ":memory:" { // allow testing
		dbpath = home
		mktables = true
	} else {
		dbpath = path.Join(home, "stmt.db")
		_, err := os.Stat(dbpath)
		switch {
		case os.IsNotExist(err):
			mktables = true
		case err != nil:
			return err
		}
	}

	err := sdb.openDB(dbpath)
	if err != nil {
		return err
	}

	if mktables {
		err = sdb.createTables()
		if err != nil {
			return err
		}
	}

	return sdb.prepareStatements()
}

func (sdb *SQLiteDB) openDB(dbpath string) error {
	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return err
	}

	sdb.db = db
	return nil
}
