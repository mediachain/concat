package main

import (
	"context"
	"database/sql"
	ggproto "github.com/gogo/protobuf/proto"
	sqlite3 "github.com/mattn/go-sqlite3"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"os"
	"path"
	"sync"
)

type SQLDB struct {
	db                 *sql.DB
	insertStmtData     *sql.Stmt
	insertStmtEnvelope *sql.Stmt
	insertStmtRefs     *sql.Stmt
	selectStmtData     *sql.Stmt
	deleteStmtData     *sql.Stmt
	deleteStmtEnvelope *sql.Stmt
	deleteStmtRefs     *sql.Stmt
	wlock              sync.Mutex
}

func (sdb *SQLDB) Put(stmt *pb.Statement) error {
	sdb.wlock.Lock()
	defer sdb.wlock.Unlock()

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
	_, err = xstmt.Exec(stmt.Id, stmt.Namespace, stmt.Publisher, mcq.StatementSource(stmt), stmt.Timestamp)
	if err != nil {
		tx.Rollback()
		return err
	}

	xstmt = tx.Stmt(sdb.insertStmtRefs)
	for wki, _ := range mcq.StatementRefs(stmt) {
		_, err = xstmt.Exec(stmt.Id, wki)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (sdb *SQLDB) PutBatch(stmts []*pb.Statement) error {
	sdb.wlock.Lock()
	defer sdb.wlock.Unlock()

	tx, err := sdb.db.Begin()
	if err != nil {
		return err
	}

	insertData := tx.Stmt(sdb.insertStmtData)
	insertEnvelope := tx.Stmt(sdb.insertStmtEnvelope)
	insertRefs := tx.Stmt(sdb.insertStmtRefs)

	for _, stmt := range stmts {
		bytes, err := ggproto.Marshal(stmt)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = insertData.Exec(stmt.Id, bytes)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = insertEnvelope.Exec(stmt.Id, stmt.Namespace, stmt.Publisher, mcq.StatementSource(stmt), stmt.Timestamp)
		if err != nil {
			tx.Rollback()
			return err
		}

		for wki, _ := range mcq.StatementRefs(stmt) {
			_, err = insertRefs.Exec(stmt.Id, wki)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
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
				sendStreamError(ctx, ch, err.Error())
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

func (sdb *SQLDB) QueryOne(q *mcq.Query) (interface{}, error) {
	sq, rsel, err := mcq.CompileQuery(q)
	if err != nil {
		return nil, err
	}

	row := sdb.db.QueryRow(sq)
	res, err := rsel.Scan(row)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoResult
		}
		return nil, err
	}

	return res, nil
}

func (sdb *SQLDB) Delete(q *mcq.Query) (count int, err error) {
	if q.Op != mcq.OpDelete {
		return 0, BadQuery
	}

	sdb.wlock.Lock()
	defer sdb.wlock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := sdb.QueryStream(ctx, q)
	if err != nil {
		return 0, err
	}

	tx, err := sdb.db.Begin()
	if err != nil {
		return 0, err
	}

	delData := tx.Stmt(sdb.deleteStmtData)
	delEnvelope := tx.Stmt(sdb.deleteStmtEnvelope)
	delRefs := tx.Stmt(sdb.deleteStmtRefs)

	for val := range ch {
		switch id := val.(type) {
		case string:
			_, err = delData.Exec(id)
			if err != nil {
				tx.Rollback()
				return 0, err
			}

			_, err = delEnvelope.Exec(id)
			if err != nil {
				tx.Rollback()
				return 0, err
			}

			_, err = delRefs.Exec(id)
			if err != nil {
				tx.Rollback()
				return 0, err
			}

			count += 1

		case StreamError:
			tx.Rollback()
			return 0, id

		default:
			tx.Rollback()
			return 0, BadResult
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (sdb *SQLDB) Close() error {
	return sdb.db.Close()
}

func (sdb *SQLDB) createTables() error {
	_, err := sdb.db.Exec("CREATE TABLE Statement (id VARCHAR(128) PRIMARY KEY, data VARBINARY)")
	if err != nil {
		return err
	}

	_, err = sdb.db.Exec("CREATE TABLE Envelope (counter INTEGER PRIMARY KEY AUTOINCREMENT, id VARCHAR(128), namespace VARCHAR, publisher VARCHAR, source VARCHAR, timestamp INTEGER)")
	if err != nil {
		return err
	}

	_, err = sdb.db.Exec("CREATE UNIQUE INDEX EnvelopeId ON Envelope (id)")
	if err != nil {
		return err
	}

	_, err = sdb.db.Exec("CREATE INDEX EnvelopeNS ON Envelope (namespace)")
	if err != nil {
		return err
	}

	_, err = sdb.db.Exec("CREATE TABLE Refs (id VARCHAR(128), wki VARCHAR)")
	if err != nil {
		return err
	}

	_, err = sdb.db.Exec("CREATE INDEX RefsId ON Refs (id)")
	if err != nil {
		return err
	}

	_, err = sdb.db.Exec("CREATE INDEX RefsWki ON Refs (wki)")
	return err
}

func (sdb *SQLDB) prepareStatements() error {
	stmt, err := sdb.db.Prepare("INSERT INTO Statement VALUES (?, ?)")
	if err != nil {
		return err
	}
	sdb.insertStmtData = stmt

	stmt, err = sdb.db.Prepare("INSERT INTO Envelope VALUES (NULL, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	sdb.insertStmtEnvelope = stmt

	stmt, err = sdb.db.Prepare("INSERT INTO Refs VALUES (?, ?)")
	if err != nil {
		return err
	}
	sdb.insertStmtRefs = stmt

	stmt, err = sdb.db.Prepare("SELECT data FROM Statement WHERE id = ?")
	if err != nil {
		return err
	}
	sdb.selectStmtData = stmt

	stmt, err = sdb.db.Prepare("DELETE FROM Statement WHERE id = ?")
	if err != nil {
		return err
	}
	sdb.deleteStmtData = stmt

	stmt, err = sdb.db.Prepare("DELETE FROM Envelope WHERE id = ?")
	if err != nil {
		return err
	}
	sdb.deleteStmtEnvelope = stmt

	stmt, err = sdb.db.Prepare("DELETE FROM Refs WHERE id = ?")
	if err != nil {
		return err
	}
	sdb.deleteStmtRefs = stmt

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
		dbdir := path.Join(home, "stmt")
		err := os.MkdirAll(dbdir, 0755)
		if err != nil {
			return err
		}

		dbpath = path.Join(dbdir, "stmt.db")
		_, err = os.Stat(dbpath)
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

		err = sdb.tuneDB()
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

func (sdb *SQLiteDB) tuneDB() error {
	_, err := sdb.db.Exec("PRAGMA journal_mode=WAL")
	return err
}

func (sdb *SQLiteDB) Merge(stmt *pb.Statement) (bool, error) {
	err := sdb.Put(stmt)
	if err != nil {
		xerr, ok := err.(sqlite3.Error)
		if ok && xerr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (sdb *SQLiteDB) MergeBatch(stmts []*pb.Statement) (count int, err error) {
	sdb.wlock.Lock()
	defer sdb.wlock.Unlock()

	tx, err := sdb.db.Begin()
	if err != nil {
		return 0, err
	}

	insertData := tx.Stmt(sdb.insertStmtData)
	insertEnvelope := tx.Stmt(sdb.insertStmtEnvelope)
	insertRefs := tx.Stmt(sdb.insertStmtRefs)

	for _, stmt := range stmts {
		bytes, err := ggproto.Marshal(stmt)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		_, err = insertData.Exec(stmt.Id, bytes)
		if err != nil {
			xerr, ok := err.(sqlite3.Error)
			if ok && xerr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				continue
			}
			tx.Rollback()
			return 0, err
		}

		_, err = insertEnvelope.Exec(stmt.Id, stmt.Namespace, stmt.Publisher, mcq.StatementSource(stmt), stmt.Timestamp)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		for wki, _ := range mcq.StatementRefs(stmt) {
			_, err = insertRefs.Exec(stmt.Id, wki)
			if err != nil {
				tx.Rollback()
				return 0, err
			}
		}

		count += 1
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return count, nil
}
