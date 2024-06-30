package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jmakaron/compman/internal/app/compman/store"
	"github.com/jmakaron/compman/internal/app/compman/types"
)

var (
	ErrUnsupportedType = errors.New("unsupported type")
	ErrNotFound        = errors.New("not found")
	ErrMissingArg      = errors.New("missing argument")
)

const (
	companiesTable string = "companies"

	colId          string = "id"
	colName        string = "name"
	colDesc        string = "description"
	colEmployeeCnt string = "employee_count"
	colRegistered  string = "registered"
	colCType       string = "company_type"
)

type companyEntity struct {
	st   *pgStore
	buff strings.Builder
	qa   []interface{}
	val  []*types.Company
	ql   []store.QueryLogEntry
}

func (e *companyEntity) logQuery(qs string, qa []interface{}, start time.Time, end time.Time) {
	if e.ql == nil {
		e.ql = []store.QueryLogEntry{}
	}
	e.ql = append(e.ql, store.QueryLogEntry{qs, qa, start, end})
}

func (e *companyEntity) QueryLog() []store.QueryLogEntry {
	return e.ql
}

func (e *companyEntity) reset() {
	e.buff.Reset()
	e.qa = []interface{}{}
	e.val = []*types.Company{}
}

func (e *companyEntity) exec(ctx context.Context) error {
	tnow := time.Now()
	conn, err := e.st.p.Acquire(e.st.ctx)
	if err != nil {
		return err
	}
	defer func() {
		conn.Release()
		if err == nil {
			e.logQuery(e.buff.String(), e.qa, tnow, time.Now())
		}
	}()
	if len(e.qa) > 0 {
		_, err = conn.Exec(ctx, e.buff.String(), e.qa...)
	} else {
		_, err = conn.Exec(ctx, e.buff.String())
	}
	return err
}

func (e *companyEntity) queryRow(ctx context.Context) error {
	tnow := time.Now()
	conn, err := e.st.p.Acquire(e.st.ctx)
	if err != nil {
		return err
	}
	defer func() {
		conn.Release()
		if err == nil || errors.Is(err, ErrNotFound) {
			e.logQuery(e.buff.String(), e.qa, tnow, time.Now())
		}
	}()
	row := conn.QueryRow(ctx, e.buff.String(), e.qa...)
	var id uuid.UUID
	var c types.Company
	if err := row.Scan(&id, &c.Name, &c.Desc, &c.EmployeeCnt, &c.Registered, &c.CType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	c.ID = id.String()
	e.val = []*types.Company{&c}
	return nil
}

func (e *companyEntity) parseRows(rows pgx.Rows) error {
	e.val = []*types.Company{}
	for rows.Next() {
		var id uuid.UUID
		var c types.Company
		if err := rows.Scan(&id, &c.Name, &c.Desc, &c.EmployeeCnt, &c.Registered, &c.CType); err != nil {
			return err
		}
		c.ID = id.String()
		e.val = append(e.val, &c)
	}
	return nil
}

func (e *companyEntity) query(ctx context.Context) error {
	tnow := time.Now()
	conn, err := e.st.p.Acquire(e.st.ctx)
	if err != nil {
		return err
	}
	defer func() {
		conn.Release()
		if err == nil || errors.Is(err, ErrNotFound) {
			e.logQuery(e.buff.String(), e.qa, tnow, time.Now())
		}
	}()
	var rows pgx.Rows
	if len(e.qa) > 0 {
		rows, err = conn.Query(ctx, e.buff.String(), e.qa...)
	} else {
		rows, err = conn.Query(ctx, e.buff.String())
	}
	if err == nil {
		err = e.parseRows(rows)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
	}
	rows.Close()
	return err
}

func (e *companyEntity) PrepareInsert(v interface{}) error {
	var err error
	e.val = []*types.Company{}
	e.qa = make([]interface{}, 6)
	e.buff.Reset()
	fmt.Fprintf(&e.buff, "INSERT INTO %s VALUES ($1, $2, $3, $4, $5, $6);",
		companiesTable)
	var c types.Company
	switch t := v.(type) {
	case []byte:
		err = json.Unmarshal(t, &c)

	default:
		err = ErrUnsupportedType
	}
	if err != nil {
		e.reset()
		return err
	}
	e.qa[0] = c.ID
	e.qa[1] = c.Name
	e.qa[2] = c.Desc
	e.qa[3] = c.EmployeeCnt
	e.qa[4] = c.Registered
	e.qa[5] = c.CType
	return nil
}

func (e *companyEntity) Insert(ctx context.Context) error {
	if e.st == nil {
		return store.ErrNotConnected
	}
	return e.exec(ctx)
}

func (e *companyEntity) PrepareSelect(v interface{}) error {
	var err error
	e.val = []*types.Company{}
	e.qa = []interface{}{}
	m := map[string]interface{}{}
	switch t := v.(type) {
	case map[string]interface{}:
		m = t
	case []byte:
		err = json.Unmarshal(t, &m)
	default:
		err = ErrUnsupportedType
	}
	if err != nil {
		e.reset()
		return err
	}
	e.buff.Reset()
	fmt.Fprintf(&e.buff, "SELECT * FROM %s", companiesTable)
	if id, ok := m[colId]; ok {
		fmt.Fprintf(&e.buff, " WHERE %s=$1", colId)
		e.qa = append(e.qa, id)
	}
	fmt.Fprintf(&e.buff, ";")
	return nil
}

func (e *companyEntity) Select(ctx context.Context) error {
	if e.st == nil {
		return store.ErrNotConnected
	}
	return e.query(ctx)
}

func (e *companyEntity) PrepareUpdate(v interface{}) error {
	var err error
	e.qa = []interface{}{}
	switch t := v.(type) {
	case map[string]interface{}:
		{
			var i interface{}
			var ok bool
			if i, ok = t[colId]; ok {
				e.buff.Reset()
				fmt.Fprintf(&e.buff, "UPDATE %s SET ", companiesTable)
				cols := []string{}
				var idx int
				for k, v := range t {
					if k == colId {
						continue
					}
					cols = append(cols, fmt.Sprintf("%s=$%d", k, idx+1))
					idx++
					e.qa = append(e.qa, v)
				}
				fmt.Fprintf(&e.buff, "%s WHERE %s=$%d RETURNING *;", strings.Join(cols, ","), colId, idx+1)
				e.qa = append(e.qa, i.(string))
				if len(e.qa) == 1 {
					err = ErrMissingArg
				}
			} else {
				err = ErrMissingArg
			}
		}
	default:
		err = ErrUnsupportedType
	}
	if err != nil {
		e.reset()
		return err
	}
	return nil
}

func (e *companyEntity) Update(ctx context.Context) error {
	if e.st == nil {
		return store.ErrNotConnected
	}
	return e.queryRow(ctx)
}

func (e *companyEntity) PrepareDelete(v interface{}) error {
	var err error
	e.qa = []interface{}{}
	switch t := v.(type) {
	case map[string]interface{}:
		if i, ok := t[colId]; ok {
			e.buff.Reset()
			fmt.Fprintf(&e.buff, "DELETE FROM %s WHERE %s=$1;", companiesTable, colId)
			e.qa = append(e.qa, i.(string))
		} else {
			err = ErrMissingArg
		}
	default:
		err = ErrUnsupportedType
	}
	if err != nil {
		e.reset()
		return err
	}
	return nil
}

func (e *companyEntity) Delete(ctx context.Context) error {
	if e.st == nil {
		return store.ErrNotConnected
	}
	return e.exec(ctx)
}

func (e *companyEntity) Value() (interface{}, error) {
	return e.val, nil
}
