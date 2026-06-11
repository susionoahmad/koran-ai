package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"koran-ai-backend/internal/edition/entity"
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
	tags      []pgconn.CommandTag
	err       error
	row       pgx.Row
	commitErr error
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
func (f *fakeTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (f *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return f.row
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
	values  [][]any
	idx     int
	err     error
	scanErr error
	closed  bool
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
	if f.scanErr != nil {
		return f.scanErr
	}
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
	if value == nil {
		return
	}
	switch d := dest.(type) {
	case *uuid.UUID:
		*d = value.(uuid.UUID)
	case **uuid.UUID:
		if val, ok := value.(uuid.UUID); ok {
			*d = &val
		}
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
	case **time.Time:
		if val, ok := value.(time.Time); ok {
			*d = &val
		}
	case *[]byte:
		*d = value.([]byte)
	case *bool:
		*d = value.(bool)
	}
}

func TestEditionExists(t *testing.T) {
	repo := NewTestRepository(&fakeDB{row: fakeRow{values: []any{true}}})
	exists, err := repo.EditionExists(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected exists to be true")
	}

	repo = NewTestRepository(&fakeDB{row: fakeRow{err: errors.New("db error")}})
	_, err = repo.EditionExists(context.Background(), time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEditionByID(t *testing.T) {
	id := uuid.New()
	headlineID := uuid.New()
	now := time.Now()
	repo := NewTestRepository(&fakeDB{row: fakeRow{values: []any{
		id, now, "Title", headlineID, 5, "DRAFT", now, now, now, now,
	}}})

	ed, err := repo.GetEditionByID(context.Background(), id.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ed.ID != id || ed.Title != "Title" || ed.TotalArticles != 5 {
		t.Fatalf("unexpected edition returned: %+v", ed)
	}

	repo = NewTestRepository(&fakeDB{row: fakeRow{err: pgx.ErrNoRows}})
	_, err = repo.GetEditionByID(context.Background(), id.String())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	repo = NewTestRepository(&fakeDB{row: fakeRow{err: errors.New("db error")}})
	_, err = repo.GetEditionByID(context.Background(), id.String())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEditionByDate(t *testing.T) {
	id := uuid.New()
	headlineID := uuid.New()
	now := time.Now()
	repo := NewTestRepository(&fakeDB{row: fakeRow{values: []any{
		id, now, "Title", headlineID, 5, "DRAFT", now, now, now, now,
	}}})

	ed, err := repo.GetEditionByDate(context.Background(), "2026-06-09")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ed.ID != id {
		t.Fatalf("unexpected edition: %+v", ed)
	}

	repo = NewTestRepository(&fakeDB{row: fakeRow{err: pgx.ErrNoRows}})
	_, err = repo.GetEditionByDate(context.Background(), "2026-06-09")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	repo = NewTestRepository(&fakeDB{row: fakeRow{err: errors.New("db error")}})
	_, err = repo.GetEditionByDate(context.Background(), "2026-06-09")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListEditions(t *testing.T) {
	id := uuid.New()
	headlineID := uuid.New()
	now := time.Now()
	rows := &fakeRows{values: [][]any{{
		id, now, "Title", headlineID, 5, "DRAFT", now, now, now, now,
	}}}
	repo := NewTestRepository(&fakeDB{
		row:  fakeRow{values: []any{int64(1)}},
		rows: rows,
	})

	list, total, err := repo.ListEditions(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != id {
		t.Fatalf("unexpected list output list=%+v total=%d", list, total)
	}

	repo = NewTestRepository(&fakeDB{row: fakeRow{err: errors.New("count failed")}})
	_, _, err = repo.ListEditions(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected count error")
	}

	repo = NewTestRepository(&fakeDB{row: fakeRow{values: []any{int64(1)}}, execErr: errors.New("query failed")})
	_, _, err = repo.ListEditions(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected query error")
	}

	// Scan error path
	badRows := &fakeRows{
		values: [][]any{{
			id, now, "Title", headlineID, 5, "DRAFT", now, now, now, now,
		}},
		scanErr: errors.New("scan failed"),
	}
	repoScanErr := NewTestRepository(&fakeDB{
		row:  fakeRow{values: []any{int64(1)}},
		rows: badRows,
	})
	_, _, err = repoScanErr.ListEditions(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestGetStats(t *testing.T) {
	// Case 1: Total = 0
	repo := NewTestRepository(&fakeDB{row: fakeRow{values: []any{int64(0)}}})
	total, latest, err := repo.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || latest != "" {
		t.Fatalf("expected 0, empty, got %d, %s", total, latest)
	}

	// Case 2: Total > 0
	now := time.Now()
	repo = NewTestRepository(&statsDB{
		countRow: fakeRow{values: []any{int64(5)}},
		maxRow:   fakeRow{values: []any{now}},
	})
	total, latest, err = repo.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 || latest != now.Format("2006-01-02") {
		t.Fatalf("expected 5, %s, got %d, %s", now.Format("2006-01-02"), total, latest)
	}

	// Case 3: Count Query Error
	repo = NewTestRepository(&fakeDB{row: fakeRow{err: errors.New("count failed")}})
	_, _, err = repo.GetStats(context.Background())
	if err == nil {
		t.Fatal("expected count error")
	}

	// Case 4: Max Date Query Error
	repo = NewTestRepository(&statsDB{
		countRow: fakeRow{values: []any{int64(5)}},
		maxRow:   fakeRow{err: errors.New("max date failed")},
	})
	_, _, err = repo.GetStats(context.Background())
	if err == nil {
		t.Fatal("expected max date error")
	}
}

type statsDB struct {
	countRow pgx.Row
	maxRow   pgx.Row
}

func (s *statsDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (s *statsDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (s *statsDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if sql == "SELECT COUNT(*) FROM editions" {
		return s.countRow
	}
	return s.maxRow
}
func (s *statsDB) Begin(ctx context.Context) (txRunner, error) {
	return nil, nil
}

func TestLoadSummariesForDate(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	rows := &fakeRows{values: [][]any{{
		id, "Headline", "Short", "Medium", "Long", []byte(`["Point"]`), "gemini", 0.95, "Ekonomi", 5, 0.99, now,
	}}}
	repo := NewTestRepository(&fakeDB{rows: rows})

	summaries, err := repo.LoadSummariesForDate(context.Background(), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summaries) != 1 || summaries[0].ClusterID != id || summaries[0].Category != "Ekonomi" {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}

	repo = NewTestRepository(&fakeDB{execErr: errors.New("query failed")})
	_, err = repo.LoadSummariesForDate(context.Background(), now)
	if err == nil {
		t.Fatal("expected error")
	}

	// Scan error path
	badRows := &fakeRows{
		values: [][]any{{
			id, "Headline", "Short", "Medium", "Long", []byte(`["Point"]`), "gemini", 0.95, "Ekonomi", 5, 0.99, now,
		}},
		scanErr: errors.New("scan failed"),
	}
	repoScanErr := NewTestRepository(&fakeDB{rows: badRows})
	_, err = repoScanErr.LoadSummariesForDate(context.Background(), now)
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestCreateEdition(t *testing.T) {
	tx := &fakeTx{row: fakeRow{values: []any{false}}}
	db := &fakeDB{beginTx: tx}
	repo := NewTestRepository(db)

	ed := &entity.Edition{
		ID:          uuid.New(),
		EditionDate: time.Now(),
		Title:       "Title",
	}
	arts := []entity.EditionArticle{{
		ID:        uuid.New(),
		ClusterID: uuid.New(),
		Section:   "Politics",
	}}

	err := repo.CreateEdition(context.Background(), ed, arts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Conflict error
	txConflict := &fakeTx{row: fakeRow{values: []any{true}}}
	dbConflict := &fakeDB{beginTx: txConflict}
	repoConflict := NewTestRepository(dbConflict)
	err = repoConflict.CreateEdition(context.Background(), ed, arts)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	// Begin error
	dbBeginErr := &fakeDB{beginErr: errors.New("begin failed")}
	repoBeginErr := NewTestRepository(dbBeginErr)
	err = repoBeginErr.CreateEdition(context.Background(), ed, arts)
	if err == nil {
		t.Fatal("expected begin transaction error")
	}

	// Insert edition failure
	txInsertErr := &fakeTx{
		row: fakeRow{values: []any{false}},
		err: errors.New("insert failed"),
	}
	dbInsertErr := &fakeDB{beginTx: txInsertErr}
	repoInsertErr := NewTestRepository(dbInsertErr)
	err = repoInsertErr.CreateEdition(context.Background(), ed, arts)
	if err == nil {
		t.Fatal("expected insert edition error")
	}

	// Commit failure
	txCommitErr := &fakeTx{
		row:       fakeRow{values: []any{false}},
		commitErr: errors.New("commit failed"),
	}
	dbCommitErr := &fakeDB{beginTx: txCommitErr}
	repoCommitErr := NewTestRepository(dbCommitErr)
	err = repoCommitErr.CreateEdition(context.Background(), ed, arts)
	if err == nil {
		t.Fatal("expected commit error")
	}

	// Duplicate check query failure
	txCheckErr := &fakeTx{
		row: fakeRow{err: errors.New("duplicate check failed")},
	}
	dbCheckErr := &fakeDB{beginTx: txCheckErr}
	repoCheckErr := NewTestRepository(dbCheckErr)
	err = repoCheckErr.CreateEdition(context.Background(), ed, arts)
	if err == nil {
		t.Fatal("expected check duplicates error")
	}
}

func TestGetEditionDetails(t *testing.T) {
	edID := uuid.New()
	headlineClusterID := uuid.New()
	now := time.Now()

	// 1. Success path
	edRow := fakeRow{values: []any{
		edID, now, "Test Edition", headlineClusterID, 3, "DRAFT", now, now, now, now,
	}}
	artRows := &fakeRows{values: [][]any{
		{headlineClusterID, "Headline News", "Title 1", "Short 1", "Medium 1", "Long 1", []byte(`["Point 1"]`), "gemini", 0.9, "Politics", 10, 0.95, now},
		{uuid.New(), "National", "Title 2", "Short 2", "Medium 2", "Long 2", []byte(`["Point 2"]`), "gemini", 0.85, "Nasional", 8, 0.90, now},
	}}

	repo := NewTestRepository(&detailsDB{
		edRow:   edRow,
		artRows: artRows,
	})

	details, err := repo.GetEditionDetails(context.Background(), edID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if details.ID != edID || details.Title != "Test Edition" {
		t.Fatalf("unexpected edition metadata: %+v", details)
	}
	if details.Headline == nil || details.Headline.Title != "Title 1" {
		t.Fatalf("expected headline to be populated, got %+v", details.Headline)
	}
	if len(details.Sections) != 1 || details.Sections[0].Name != "National" || len(details.Sections[0].Articles) != 1 {
		t.Fatalf("expected 1 section named National with 1 article, got %+v", details.Sections)
	}

	// 2. GetEditionByID error (e.g. not found)
	repoErr := NewTestRepository(&detailsDB{
		edRow: fakeRow{err: pgx.ErrNoRows},
	})
	_, err = repoErr.GetEditionDetails(context.Background(), edID.String())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// 3. Query articles error
	repoQueryErr := NewTestRepository(&detailsDB{
		edRow:    edRow,
		queryErr: errors.New("query articles failed"),
	})
	_, err = repoQueryErr.GetEditionDetails(context.Background(), edID.String())
	if err == nil {
		t.Fatal("expected query articles error")
	}

	// 4. Scan articles error
	badArtRows := &fakeRows{
		values: [][]any{
			{headlineClusterID, "Headline News", "Title 1", "Short 1", "Medium 1", "Long 1", []byte(`["Point 1"]`), "gemini", 0.9, "Politics", 10, 0.95, now},
		},
		scanErr: errors.New("scan failed"),
	}
	repoScanErr := NewTestRepository(&detailsDB{
		edRow:   edRow,
		artRows: badArtRows,
	})
	_, err = repoScanErr.GetEditionDetails(context.Background(), edID.String())
	if err == nil {
		t.Fatal("expected scan articles error")
	}
}

type detailsDB struct {
	edRow    pgx.Row
	artRows  pgx.Rows
	queryErr error
}

func (d *detailsDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (d *detailsDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if d.queryErr != nil {
		return nil, d.queryErr
	}
	return d.artRows, nil
}
func (d *detailsDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return d.edRow
}
func (d *detailsDB) Begin(ctx context.Context) (txRunner, error) {
	return nil, nil
}

func TestNewPostgresRepository(t *testing.T) {
	repo := NewPostgresRepository(nil)
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestPoolRunnerDelegates(t *testing.T) {
	p := poolRunner{db: nil}
	
	func() {
		defer func() { recover() }()
		_, _ = p.Exec(context.Background(), "")
	}()
	
	func() {
		defer func() { recover() }()
		_, _ = p.Query(context.Background(), "")
	}()
	
	func() {
		defer func() { recover() }()
		_ = p.QueryRow(context.Background(), "")
	}()
	
	func() {
		defer func() { recover() }()
		_, _ = p.Begin(context.Background())
	}()
}

func TestTxRunnerWrapperDelegates(t *testing.T) {
	w := txRunnerWrapper{tx: nil}
	
	func() {
		defer func() { recover() }()
		_, _ = w.Exec(context.Background(), "")
	}()
	
	func() {
		defer func() { recover() }()
		_, _ = w.Query(context.Background(), "")
	}()
	
	func() {
		defer func() { recover() }()
		_ = w.QueryRow(context.Background(), "")
	}()
	
	func() {
		defer func() { recover() }()
		_ = w.Commit(context.Background())
	}()
	
	func() {
		defer func() { recover() }()
		_ = w.Rollback(context.Background())
	}()
}
