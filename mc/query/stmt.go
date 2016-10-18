package query

import (
	pb "github.com/mediachain/concat/proto"
)

func StatementRefs(stmt *pb.Statement) []string {
	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Simple:
		return body.Simple.Refs

	default:
		// TODO compound statements etc
		return nil
	}
}

func StatementSource(stmt *pb.Statement) string {
	// TODO compound statements etc
	return stmt.Publisher
}
