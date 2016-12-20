package query

import (
	pb "github.com/mediachain/concat/proto"
)

func StatementRefs(stmt *pb.Statement) []string {
	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Simple:
		return body.Simple.Refs

	case *pb.StatementBody_Compound:
		refs := makeStatementRefSet()
		refs.mergeCompound(body.Compound)
		return refs.toList()

	case *pb.StatementBody_Envelope:
		refs := makeStatementRefSet()
		refs.mergeEnvelope(body.Envelope)
		return refs.toList()

	default:
		return nil
	}
}

type StatementRefSet map[string]bool

func makeStatementRefSet() StatementRefSet {
	return StatementRefSet(make(map[string]bool))
}

func (refs StatementRefSet) mergeStatement(stmt *pb.Statement) {
	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Simple:
		refs.mergeSimple(body.Simple)

	case *pb.StatementBody_Compound:
		refs.mergeCompound(body.Compound)

	case *pb.StatementBody_Envelope:
		refs.mergeEnvelope(body.Envelope)
	}
}

func (refs StatementRefSet) mergeSimple(stmt *pb.SimpleStatement) {
	for _, wki := range stmt.Refs {
		refs[wki] = true
	}
}

func (refs StatementRefSet) mergeCompound(stmt *pb.CompoundStatement) {
	for _, xstmt := range stmt.Body {
		refs.mergeSimple(xstmt)
	}
}

func (refs StatementRefSet) mergeEnvelope(stmt *pb.EnvelopeStatement) {
	for _, xstmt := range stmt.Body {
		refs.mergeStatement(xstmt)
	}
}

func (refs StatementRefSet) toList() []string {
	lst := make([]string, len(refs))
	x := 0
	for wki, _ := range refs {
		lst[x] = wki
		x++
	}
	return lst
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
