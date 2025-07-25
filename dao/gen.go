// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package dao

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"gorm.io/gen"

	"gorm.io/plugin/dbresolver"
)

var (
	Q                  = new(Query)
	DataDate           *dataDate
	DredgerDataHl      *dredgerDataHl
	DredgerDatum       *dredgerDatum
	TheoryOptimalParam *theoryOptimalParam
)

func SetDefault(db *gorm.DB, opts ...gen.DOOption) {
	*Q = *Use(db, opts...)
	DataDate = &Q.DataDate
	DredgerDataHl = &Q.DredgerDataHl
	DredgerDatum = &Q.DredgerDatum
	TheoryOptimalParam = &Q.TheoryOptimalParam
}

func Use(db *gorm.DB, opts ...gen.DOOption) *Query {
	return &Query{
		db:                 db,
		DataDate:           newDataDate(db, opts...),
		DredgerDataHl:      newDredgerDataHl(db, opts...),
		DredgerDatum:       newDredgerDatum(db, opts...),
		TheoryOptimalParam: newTheoryOptimalParam(db, opts...),
	}
}

type Query struct {
	db *gorm.DB

	DataDate           dataDate
	DredgerDataHl      dredgerDataHl
	DredgerDatum       dredgerDatum
	TheoryOptimalParam theoryOptimalParam
}

func (q *Query) Available() bool { return q.db != nil }

func (q *Query) clone(db *gorm.DB) *Query {
	return &Query{
		db:                 db,
		DataDate:           q.DataDate.clone(db),
		DredgerDataHl:      q.DredgerDataHl.clone(db),
		DredgerDatum:       q.DredgerDatum.clone(db),
		TheoryOptimalParam: q.TheoryOptimalParam.clone(db),
	}
}

func (q *Query) ReadDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Read))
}

func (q *Query) WriteDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Write))
}

func (q *Query) ReplaceDB(db *gorm.DB) *Query {
	return &Query{
		db:                 db,
		DataDate:           q.DataDate.replaceDB(db),
		DredgerDataHl:      q.DredgerDataHl.replaceDB(db),
		DredgerDatum:       q.DredgerDatum.replaceDB(db),
		TheoryOptimalParam: q.TheoryOptimalParam.replaceDB(db),
	}
}

type queryCtx struct {
	DataDate           IDataDateDo
	DredgerDataHl      IDredgerDataHlDo
	DredgerDatum       IDredgerDatumDo
	TheoryOptimalParam ITheoryOptimalParamDo
}

func (q *Query) WithContext(ctx context.Context) *queryCtx {
	return &queryCtx{
		DataDate:           q.DataDate.WithContext(ctx),
		DredgerDataHl:      q.DredgerDataHl.WithContext(ctx),
		DredgerDatum:       q.DredgerDatum.WithContext(ctx),
		TheoryOptimalParam: q.TheoryOptimalParam.WithContext(ctx),
	}
}

func (q *Query) Transaction(fc func(tx *Query) error, opts ...*sql.TxOptions) error {
	return q.db.Transaction(func(tx *gorm.DB) error { return fc(q.clone(tx)) }, opts...)
}

func (q *Query) Begin(opts ...*sql.TxOptions) *QueryTx {
	tx := q.db.Begin(opts...)
	return &QueryTx{Query: q.clone(tx), Error: tx.Error}
}

type QueryTx struct {
	*Query
	Error error
}

func (q *QueryTx) Commit() error {
	return q.db.Commit().Error
}

func (q *QueryTx) Rollback() error {
	return q.db.Rollback().Error
}

func (q *QueryTx) SavePoint(name string) error {
	return q.db.SavePoint(name).Error
}

func (q *QueryTx) RollbackTo(name string) error {
	return q.db.RollbackTo(name).Error
}
