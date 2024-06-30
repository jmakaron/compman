package compman

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmakaron/compman/internal/app/compman/store/postgres"
	"github.com/jmakaron/compman/internal/app/compman/types"
	httpsrv "github.com/jmakaron/compman/internal/pkg/http"
	"io"
	"net/http"
)

const (
	companyGet    = "company-get"
	companyList   = "company-list"
	companyInsert = "company-insert"
	companyDelete = "company-delete"
	companyUpdate = "company-update"
)

func (c *ServiceComponent) getRestAPI() (httpsrv.RouteLayout, *httpsrv.RouterSpec) {
	rl := httpsrv.RouteLayout{
		"/company": {
			companyGet:    {http.MethodGet, "/{id1}"},
			companyList:   {http.MethodGet, ""},
			companyInsert: {http.MethodPost, ""},
			companyDelete: {http.MethodDelete, "/{id1}"},
			companyUpdate: {http.MethodPut, "/{id1}"},
		}}
	rs := httpsrv.RouterSpec{
		companyGet:    c.companyGetHandler,
		companyList:   c.companyListHandler,
		companyInsert: c.companyInsertHandler,
		companyDelete: c.companyDeleteHandler,
		companyUpdate: c.companyUpdateHandler,
	}
	return rl, &rs

}

func (c *ServiceComponent) companyGetHandler(w http.ResponseWriter, r *http.Request) error {
	e, err := c.st.NewEntity(&types.Company{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer func() {
		for _, entry := range e.QueryLog() {
			c.log.Debug(fmt.Sprintf("[DB]: %s %+v", entry.End.Sub(entry.Start), entry))
		}
	}()
	id := httpsrv.GetIdList(r)[0]
	if err := uuid.Validate(id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	if err := e.PrepareSelect(map[string]interface{}{
		"id": id,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if err := e.Select(context.Background()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	var v interface{}
	v, err = e.Value()
	if err != nil {
		if !errors.Is(err, postgres.ErrNotFound) {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return err
	}
	rv := v.([]*types.Company)[0]
	b, err := json.Marshal(rv)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return nil
}

func (c *ServiceComponent) companyListHandler(w http.ResponseWriter, r *http.Request) error {
	e, err := c.st.NewEntity(&types.Company{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer func() {
		for _, entry := range e.QueryLog() {
			c.log.Debug(fmt.Sprintf("[DB]: %s %+v", entry.End.Sub(entry.Start), entry))
		}
	}()
	if err := e.PrepareSelect(map[string]interface{}{}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	var rv []*types.Company
	if err := e.Select(context.Background()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	var i interface{}
	i, err = e.Value()
	if err != nil {
		if !errors.Is(err, postgres.ErrNotFound) {
			w.WriteHeader(http.StatusInternalServerError)
			return err
		}
	} else {
		rv = i.([]*types.Company)
	}
	if len(rv) == 0 {
		rv = []*types.Company{}
	}
	b, err := json.Marshal(rv)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return nil
}

func (c *ServiceComponent) companyInsertHandler(w http.ResponseWriter, r *http.Request) error {
	var company types.Company
	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer r.Body.Close()
	err = json.Unmarshal(b, &company)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	if len(company.Name) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	if company.CType < types.CompanyTypeCorporation || company.CType > types.CompanyTypeSoleProprietorship {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	company.ID = uuid.NewString()
	b, err = json.Marshal(&company)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	e, err := c.st.NewEntity(&company)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer func() {
		for _, entry := range e.QueryLog() {
			c.log.Debug(fmt.Sprintf("[DB]: %s %+v", entry.End.Sub(entry.Start), entry))
		}
	}()
	if err = e.PrepareInsert(b); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if err = e.Insert(context.Background()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return nil
}

func (c *ServiceComponent) companyDeleteHandler(w http.ResponseWriter, r *http.Request) error {
	id := httpsrv.GetIdList(r)[0]
	if err := uuid.Validate(id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	e, err := c.st.NewEntity(&types.Company{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer func() {
		for _, entry := range e.QueryLog() {
			c.log.Debug(fmt.Sprintf("[DB]: %s %+v", entry.End.Sub(entry.Start), entry))
		}
	}()
	if err = e.PrepareDelete(map[string]interface{}{"id": id}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if err = e.Delete(context.Background()); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (c *ServiceComponent) companyUpdateHandler(w http.ResponseWriter, r *http.Request) error {
	id := httpsrv.GetIdList(r)[0]
	if err := uuid.Validate(id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	r.Body.Close()
	m := map[string]interface{}{}
	if err = json.Unmarshal(b, &m); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	{
		m["id"] = id
		if v, ok := m["type"]; ok {
			ctype := types.ParseCompanyType(v.(string))
			if ctype == -1 {
				w.WriteHeader(http.StatusBadRequest)
				return nil
			}
			m["type"] = ctype
		}
	}
	e, err := c.st.NewEntity(&types.Company{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer func() {
		for _, entry := range e.QueryLog() {
			c.log.Debug(fmt.Sprintf("[DB]: %s %+v", entry.End.Sub(entry.Start), entry))
		}
	}()
	if err = e.PrepareUpdate(m); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if err = e.Update(context.Background()); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return err
	}
	i, err := e.Value()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	b, err = json.Marshal(i.([]*types.Company)[0])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return nil
}
