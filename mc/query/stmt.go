package query

import (
	pb "github.com/mediachain/concat/proto"
)

func StatementRefs(stmt *pb.Statement) []string {
	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Simple:
		return body.Simple.Refs

	case *pb.StatementBody_Compound:
		stmts := body.Compound.Body
		count := 0
		for _, stmt := range stmts {
			count += len(stmt.Refs)
		}
		refs := make([]string, 0, count)
		for _, stmt := range stmts {
			refs = append(refs, stmt.Refs...)
		}
		return refs

	case *pb.StatementBody_Envelope:
		stmts := body.Envelope.Body
		count := 0
		for _, stmt := range stmts {
			count += countStatementRefs(stmt)
		}
		refs := make([]string, 0, count)
		for _, stmt := range stmts {
			refs = append(refs, StatementRefs(stmt)...)
		}
		return refs

	default:
		return nil
	}
}

func countStatementRefs(stmt *pb.Statement) int {
	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Simple:
		return len(body.Simple.Refs)

	case *pb.StatementBody_Compound:
		stmts := body.Compound.Body
		count := 0
		for _, stmt := range stmts {
			count += len(stmt.Refs)
		}
		return count

	case *pb.StatementBody_Envelope:
		stmts := body.Envelope.Body
		count := 0
		for _, stmt := range stmts {
			count += countStatementRefs(stmt)
		}
		return count

	default:
		return 0
	}
}

func StatementSource(stmt *pb.Statement) string {
	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Envelope:
		// that's just the source of the first statement; all envelope statements
		// should have the same source
		if len(body.Envelope.Body) > 0 {
			return StatementSource(body.Envelope.Body[0])
		}
		return stmt.Publisher

	default:
		return stmt.Publisher
	}
}
