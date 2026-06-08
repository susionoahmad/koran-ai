package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"koran-ai-backend/internal/clustering/entity"
)

type fakeDB struct {
	execTag  pgconn.CommandTag
	execErr  error
	rows     pgx.Rows
	row      pgx.Row
	beginTx  txRunner
	beginErr error
}

func (f *fakeDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return f.execTag, f.execErr
}
func (f *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if f.execErr != nil {
		return nil, f.execErr
	}
	return f.rows, nil
}
func (f *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return f.row
}
func (f *fakeDB) Begin(ctx context.Context) (txRunner, error) {
	return f.beginTx, f.beginErr
}

type fakeTx struct {
	tags       []pgconn.CommandTag
	err        error
	commitErr  error
	rolledBack bool
}

func (f *fakeTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if f.err != nil {
		return pgconn.CommandTag{}, f.err
	}
	if len(f.tags) == 0 {
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	tag := f.tags[0]
	f.tags = f.tags[1:]
	return tag, nil
}
func (f *fakeTx) Commit(ctx context.Context) error {
	return f.commitErr
}
func (f *fakeTx) Rollback(ctx context.Context) error {
	f.rolledBack = true
	return nil
}

type fakeRow struct {
	values []any
	err    error
}

func (f fakeRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	for i := range dest {
		assign(dest[i], f.values[i])
	}
	return nil
}

type fakeRows struct {
	values [][]any
	idx    int
	err    error
	closed bool
}

func (f *fakeRows) Close()                                       { f.closed = true }
func (f *fakeRows) Err() error                                   { return f.err }
func (f *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT 1") }
func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakeRows) Next() bool {
	if f.idx >= len(f.values) {
		return false
	}
	f.idx++
	return true
}
func (f *fakeRows) Scan(dest ...any) error {
	row := f.values[f.idx-1]
	for i := range dest {
		assign(dest[i], row[i])
	}
	return nil
}
func (f *fakeRows) Values() ([]any, error) { return f.values[f.idx-1], nil }
func (f *fakeRows) RawValues() [][]byte    { return nil }
func (f *fakeRows) Conn() *pgx.Conn        { return nil }

func assign(dest any, value any) {
	switch d := dest.(type) {
	case *uuid.UUID:
		*d = value.(uuid.UUID)
	case *string:
		*d = value.(string)
	case *int:
		*d = value.(int)
	case *int64:
		*d = value.(int64)
	case *float64:
		*d = value.(float64)
	case *time.Time:
		*d = value.(time.Time)
	}
}

func TestCreateCluster(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{}}
	err := repo.CreateCluster(context.Background(), &entity.Cluster{
		ID:           uuid.New(),
		Title:        "Bank Indonesia Pertahankan BI Rate",
		ArticleCount: 1,
		Confidence:   1,
	})
	if err != nil {
		t.Fatalf("CreateCluster returned error: %v", err)
	}
}

func TestCreateCluster_Error(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{execErr: errors.New("insert failed")}}
	err := repo.CreateCluster(context.Background(), &entity.Cluster{ID: uuid.New(), Title: "Title"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddArticleToCluster(t *testing.T) {
	tx := &fakeTx{tags: []pgconn.CommandTag{
		pgconn.NewCommandTag("INSERT 0 1"),
		pgconn.NewCommandTag("UPDATE 1"),
	}}
	repo := &postgresRepository{db: &fakeDB{beginTx: tx}}
	err := repo.AddArticleToCluster(context.Background(), uuid.New().String(), uuid.New().String())
	if err != nil {
		t.Fatalf("AddArticleToCluster returned error: %v", err)
	}
	if !tx.rolledBack {
		t.Fatal("expected deferred rollback to be called")
	}
}

func TestAddArticleToCluster_ArticleNotFound(t *testing.T) {
	tx := &fakeTx{tags: []pgconn.CommandTag{
		pgconn.NewCommandTag("INSERT 0 1"),
		pgconn.NewCommandTag("UPDATE 0"),
	}}
	repo := &postgresRepository{db: &fakeDB{beginTx: tx}}
	err := repo.AddArticleToCluster(context.Background(), uuid.New().String(), uuid.New().String())
	if err == nil {
		t.Fatal("expected article not found error")
	}
}

func TestAddArticleToCluster_BeginError(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{beginErr: errors.New("begin failed")}}
	err := repo.AddArticleToCluster(context.Background(), uuid.New().String(), uuid.New().String())
	if err == nil {
		t.Fatal("expected begin error")
	}
}

func TestAddArticleToCluster_ExecError(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{beginTx: &fakeTx{err: errors.New("exec failed")}}}
	err := repo.AddArticleToCluster(context.Background(), uuid.New().String(), uuid.New().String())
	if err == nil {
		t.Fatal("expected exec error")
	}
}

func TestAddArticleToCluster_CommitError(t *testing.T) {
	tx := &fakeTx{
		tags: []pgconn.CommandTag{
			pgconn.NewCommandTag("INSERT 0 1"),
			pgconn.NewCommandTag("UPDATE 1"),
		},
		commitErr: errors.New("commit failed"),
	}
	repo := &postgresRepository{db: &fakeDB{beginTx: tx}}
	err := repo.AddArticleToCluster(context.Background(), uuid.New().String(), uuid.New().String())
	if err == nil {
		t.Fatal("expected commit error")
	}
}

func TestListClusters(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	rows := &fakeRows{values: [][]any{{
		id, "Cluster title", "ekonomi", 2, 0.88, now, now,
	}}}
	repo := &postgresRepository{db: &fakeDB{
		row:  fakeRow{values: []any{int64(1)}},
		rows: rows,
	}}

	clusters, total, err := repo.ListClusters(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("ListClusters returned error: %v", err)
	}
	if total != 1 || len(clusters) != 1 || clusters[0].ID != id {
		t.Fatalf("unexpected result total=%d clusters=%+v", total, clusters)
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
}

func TestListClusters_CountError(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{row: fakeRow{err: errors.New("count failed")}}}
	_, _, err := repo.ListClusters(context.Background(), 1, 20)
	if err == nil {
		t.Fatal("expected count error")
	}
}

func TestListClusters_QueryError(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{
		row:     fakeRow{values: []any{int64(1)}},
		execErr: errors.New("query failed"),
	}}
	_, _, err := repo.ListClusters(context.Background(), 1, 20)
	if err == nil {
		t.Fatal("expected query error")
	}
}

func TestListClusters_RowsError(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{
		row:  fakeRow{values: []any{int64(0)}},
		rows: &fakeRows{err: errors.New("rows failed")},
	}}
	_, _, err := repo.ListClusters(context.Background(), 1, 20)
	if err == nil {
		t.Fatal("expected rows error")
	}
}

func TestGetClusterByID(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	repo := &postgresRepository{db: &fakeDB{row: fakeRow{values: []any{
		id, "Cluster title", "ekonomi", 2, 0.88, now, now,
	}}}}

	cluster, err := repo.GetClusterByID(context.Background(), id.String())
	if err != nil {
		t.Fatalf("GetClusterByID returned error: %v", err)
	}
	if cluster.ID != id || cluster.Title != "Cluster title" {
		t.Fatalf("unexpected cluster: %+v", cluster)
	}
}

func TestGetClusterByID_NotFound(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{row: fakeRow{err: pgx.ErrNoRows}}}
	_, err := repo.GetClusterByID(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetClusterByID_Error(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{row: fakeRow{err: errors.New("scan failed")}}}
	_, err := repo.GetClusterByID(context.Background(), uuid.New().String())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateClusterStats(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{execTag: pgconn.NewCommandTag("UPDATE 1")}}
	err := repo.UpdateClusterStats(context.Background(), uuid.New().String(), 2, 0.75)
	if err != nil {
		t.Fatalf("UpdateClusterStats returned error: %v", err)
	}
}

func TestUpdateClusterStats_NotFound(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{execTag: pgconn.NewCommandTag("UPDATE 0")}}
	err := repo.UpdateClusterStats(context.Background(), uuid.New().String(), 2, 0.75)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateClusterStats_Error(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{execErr: errors.New("update failed")}}
	err := repo.UpdateClusterStats(context.Background(), uuid.New().String(), 2, 0.75)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCountClusteredArticles(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{row: fakeRow{values: []any{int64(4)}}}}
	count, err := repo.CountClusteredArticles(context.Background())
	if err != nil {
		t.Fatalf("CountClusteredArticles returned error: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}
}

func TestCountClusteredArticles_Error(t *testing.T) {
	repo := &postgresRepository{db: &fakeDB{row: fakeRow{err: errors.New("count failed")}}}
	_, err := repo.CountClusteredArticles(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNullableString(t *testing.T) {
	if nullableString("") != nil {
		t.Fatal("expected empty string to become nil")
	}
	got := nullableString("ekonomi")
	if got == nil || *got != "ekonomi" {
		t.Fatalf("expected ekonomi pointer, got %#v", got)
	}
}
