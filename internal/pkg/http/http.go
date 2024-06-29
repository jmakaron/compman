package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/gorilla/mux"

	"github.com/jmakaron/compman/pkg/logger"
)

type loggedResp struct {
	w      http.ResponseWriter
	status int
	//err    error
}

func (lr *loggedResp) Header() http.Header {
	return lr.w.Header()
}

func (lr *loggedResp) Write(b []byte) (int, error) {
	return lr.w.Write(b)
}

func (lr *loggedResp) WriteHeader(statusCode int) {
	lr.status = statusCode
	lr.w.WriteHeader(statusCode)
}

type HandlerWithError func(http.ResponseWriter, *http.Request) error

/* first key: prefix in the format of /<noun>/id1/<noun>/id2/<verb>
 * second key: http method
 * value: func(http.ResponseWriter, *http.Request) error
 */
type RouterSpec map[string]map[string]HandlerWithError

type HTTPServiceCfg struct {
	Secure    bool
	Addr      string
	Port      int
	CertFile  string
	KeyFile   string
	SrvPrefix string
	Debug     bool
}

type HTTPService struct {
	log      *logger.Logger
	srv      *http.Server
	isSecure bool
	ctx      context.Context
	cancel   context.CancelFunc
	certFile string
	keyFile  string
	ep       string
	Scheme   string
}

func (h *HTTPService) wrapHandler(handler HandlerWithError) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srcIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		reqURL := r.URL.String()
		start := time.Now()
		var handlerErr error
		lr := &loggedResp{w: w}
		defer func() {
			var logErr error
			var logMsg string
			if r := recover(); r != nil {
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, false)
				h.log.Error(string(buf))
				logErr = fmt.Errorf("%v", r)
			} else {
				logErr = handlerErr
			}
			logMsg = fmt.Sprintf("[%s %s] %v %v %v [%v] (%v) <%#v>",
				h.Scheme, h.srv.Addr, srcIP, r.Method, reqURL,
				lr.status, time.Since(start),
				logErr)
			if handlerErr != nil {
				h.log.Error(logMsg)
			} else {
				h.log.Info(logMsg)
			}
		}()
		handlerErr = handler(lr, r)
	}
}

func (h *HTTPService) Init(cfg HTTPServiceCfg, rspec RouterSpec, log *logger.Logger) error {
	h.certFile = cfg.CertFile
	h.keyFile = cfg.KeyFile
	h.ep = fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port)
	h.log = log
	if len(h.certFile) > 0 && len(h.keyFile) > 0 {
		h.Scheme = "https"
	} else {
		h.Scheme = "http"
	}
	h.srv = &http.Server{Addr: h.ep}
	h.registerHandlers(cfg.SrvPrefix, cfg.Debug, rspec)
	return nil
}

func (h *HTTPService) registerHandlers(srvPrefix string, debug bool, rspec RouterSpec) {
	router := mux.NewRouter()
	r := router.PathPrefix(fmt.Sprintf("/%sd", srvPrefix))
	if debug {
		router.PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
		router.PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
		router.PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
		router.PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
		router.PathPrefix("/debug/pprof").HandlerFunc(pprof.Index)
	}
	for prefix, rmap := range rspec {
		var entry *mux.Router
		if prefix == "/metrics" {
			entry = router.PathPrefix(prefix).Subrouter() // map metrics to ep/metrics
		} else {
			entry = r.PathPrefix(prefix).Subrouter() // map metrics to ep/srvPrefix/<prefix>
		}
		for method, handler := range rmap {
			if handler != nil {
				entry.HandleFunc(prefix, h.wrapHandler(handler)).Methods(method)
			}
		}
	}

}

func (h *HTTPService) Start() error {
	h.ctx, h.cancel = context.WithCancel(context.Background())
	errCh := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() {
		var err error
		if h.isSecure {
			err = h.srv.ListenAndServeTLS(h.certFile, h.keyFile)
		} else {
			err = h.srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			select {
			case <-ctx.Done():
				h.log.Error(fmt.Sprintf("failed to start http server: %+v", err))
			default:
				errCh <- err
			}
		}
	}()
	var err error
	select {
	case <-ctx.Done():
	case err = <-errCh:
	}
	return err
}

func (h *HTTPService) Stop() error {
	var err error
	if h.cancel != nil {
		h.cancel()
		err = h.srv.Shutdown(context.Background())
	}
	return err
}

func GetIdList(r *http.Request) []string {
	vars := mux.Vars(r)
	l := []string{}
	if vars != nil {
		for i, ok := 1, true; ok; i++ {
			id := fmt.Sprintf("id%d", i)
			var v string
			if v, ok = vars[id]; ok {
				l = append(l, v)
			}
		}
	}
	return l
}

/*
func GetParams(r *http.Request) map[string]string {
	return nil
}
*/
