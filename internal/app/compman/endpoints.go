package compman

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/jmakaron/compman/internal/app/compman/store/postgres"
	"github.com/jmakaron/compman/internal/app/compman/types"
	httpsrv "github.com/jmakaron/compman/internal/pkg/http"
)

const (
	companyGet    = "company-get"
	companyList   = "company-list"
	companyInsert = "company-insert"
	companyDelete = "company-delete"
	companyUpdate = "company-update"
	serviceLogin  = "login"
)

func (c *ServiceComponent) getRestAPI() (httpsrv.RouteLayout, *httpsrv.RouterSpec) {
	rl := httpsrv.RouteLayout{
		"/login": {
			serviceLogin: {http.MethodPost, ""},
		},
		"/company": {
			companyGet:    {http.MethodGet, "/{id1}"},
			companyList:   {http.MethodGet, ""},
			companyInsert: {http.MethodPost, ""},
			companyDelete: {http.MethodDelete, "/{id1}"},
			companyUpdate: {http.MethodPatch, "/{id1}"},
		}}
	rs := httpsrv.RouterSpec{
		serviceLogin:  c.serviceLogin,
		companyGet:    c.companyGetHandler,
		companyList:   c.companyListHandler,
		companyInsert: httpsrv.JWTAuth(c.companyInsertHandler),
		companyDelete: httpsrv.JWTAuth(c.companyDeleteHandler),
		companyUpdate: httpsrv.JWTAuth(c.companyUpdateHandler),
	}
	return rl, &rs

}

func (c *ServiceComponent) serviceLogin(w http.ResponseWriter, r *http.Request) error {
	type loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	var lr loginReq
	err = json.Unmarshal(b, &lr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	if lr.Username != c.cfg.Username || lr.Password != c.cfg.Password {
		w.WriteHeader(http.StatusForbidden)
		return nil
	}
	secretKey := []byte("MY_SECRET_key")
	claims := jwt.MapClaims{
		"authorized": true,
		"user":       lr.Username,
		"exp":        time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
	w.WriteHeader(http.StatusOK)
	return nil
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
	var rollback bool
	defer func(company *types.Company) {
		if rollback {
			if err := e.PrepareDelete(map[string]interface{}{"id": company.ID}); err != nil {
				c.log.Error(fmt.Sprintf("failed to rollback insert operation, %+v", err))
				return
			}
			if err := e.Delete(context.Background()); err != nil {
				c.log.Error(fmt.Sprintf("failed to rollback insert operation, %+v", err))
				return
			}
		}
	}(&company)
	evt, err := types.NewKafkaCompanyEvent(&company, "insert")
	if err != nil {
		rollback = true
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if err := c.kp.PublishWithRetry(evt); err != nil {
		rollback = true
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
	var rollback bool
	i, err := e.Value()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	company := i.([]*types.Company)[0]
	defer func(company *types.Company) {
		if rollback {
			b, _ := json.Marshal(company)
			if err := e.PrepareInsert(b); err != nil {
				c.log.Error(fmt.Sprintf("failed to rollback delete operation, %+v", err))
				return
			}
			if err = e.Insert(context.Background()); err != nil {
				c.log.Error(fmt.Sprintf("failed to rollback delete operation, %+v", err))
				return
			}
		}
	}(company)
	evt, err := types.NewKafkaCompanyEvent(company, "delete")
	if err != nil {
		rollback = true
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if err := c.kp.PublishWithRetry(evt); err != nil {
		rollback = true
		w.WriteHeader(http.StatusInternalServerError)
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
		if id2, ok := m[id]; ok && id != id2 {
			w.WriteHeader(http.StatusBadRequest)
			return nil
		} else if !ok {
			m["id"] = id
		}
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
	var rollback bool
	{
		if err := e.PrepareSelect(map[string]interface{}{"id": id}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return err
		}
		if err := e.Select(context.Background()); err != nil {
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
		defer func(company *types.Company) {
			if rollback {
				m := map[string]interface{}{}
				b, _ := json.Marshal(company)
				json.Unmarshal(b, &m)
				m["type"] = company.CType
				if err = e.PrepareUpdate(m); err != nil {
					c.log.Error(fmt.Sprintf("failed to rollback update operation, %+v", err))
					return
				}
				if err = e.Update(context.Background()); err != nil {
					c.log.Error(fmt.Sprintf("failed to rollback update operation, %+v", err))
					return
				}
			}
		}(i.([]*types.Company)[0])
	}
	if err = e.PrepareUpdate(m); err != nil {
		if errors.Is(err, postgres.ErrInvalidArg) ||
			errors.Is(err, postgres.ErrMissingArg) {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
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
	company := i.([]*types.Company)[0]
	evt, err := types.NewKafkaCompanyEvent(company, "update")
	if err != nil {
		rollback = true
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if err := c.kp.PublishWithRetry(evt); err != nil {
		rollback = true
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	b, err = json.Marshal(company)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return nil
}
