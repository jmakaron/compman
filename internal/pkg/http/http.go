package http

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func JWTAuth(handler HandlerWithError) HandlerWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}
		tokenStr := strings.Replace(authHeader, "Bearer ", "", 1)
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte("MY_SECRET_key"), nil
		})
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			r = r.WithContext(context.WithValue(r.Context(), "claims", claims))
			return handler(w, r)
		}
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}
}

// first key is endpoint prefix, second key is handler name, value is [http.Method, <endpoint suffix regexp>]
type RouteLayout map[string]map[string][]string

/* key: handler name
 * value: func(http.ResponseWriter, *http.Request) error
 */
type RouterSpec map[string]HandlerWithError

type HTTPServiceCfg struct {
	Secure    bool   `json:"secure"`
	Addr      string `json:"addr"`
	Port      int    `json:"port"`
	CertFile  string `json:"cert_file,omitempty"`
	KeyFile   string `json:"key_file,omitempty"`
	SrvPrefix string `json:"service_prefix"`
	Debug     bool   `json:"-"`
}

type HTTPService struct {
	log      *logger.Logger
	srv      *http.Server
	isSecure bool
	ctx      context.Context
	cancel   context.CancelFunc
	certFile string
	keyFile  string
	Scheme   string
	ep       string
	prefix   string
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

func (h *HTTPService) Init(cfg HTTPServiceCfg, layout RouteLayout, rspec *RouterSpec, log *logger.Logger) error {
	h.log = log
	h.certFile = cfg.CertFile
	h.keyFile = cfg.KeyFile
	if cfg.Port == 0 {
		cfg.Port = rand.Intn(8998) + 1001
		h.log.Debug("cfg port is 0, choosing one at random in range (1000, 10000)")
	}
	h.ep = fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port)
	if len(h.certFile) > 0 && len(h.keyFile) > 0 {
		h.Scheme = "https"
	} else {
		h.Scheme = "http"
	}
	h.srv = &http.Server{Addr: h.ep}
	router := mux.NewRouter()
	if cfg.Debug {
		router.PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
		router.PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
		router.PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
		router.PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
		router.PathPrefix("/debug/pprof").HandlerFunc(pprof.Index)
	}
	r := router.PathPrefix(fmt.Sprintf("/%s", cfg.SrvPrefix)).Subrouter()
	for prefix, api := range layout {
		entry := r.PathPrefix(prefix).Subrouter()
		for name, apiData := range api {
			if handler := (*rspec)[name]; handler != nil {
				entry.HandleFunc(apiData[1], h.wrapHandler(handler)).Methods(apiData[0]).Name(name)
				h.log.Debug(fmt.Sprintf("registered %s: %s %s%s", name, apiData[0], prefix, apiData[1]))
			}
		}
	}
	h.srv.Handler = router
	h.prefix = cfg.SrvPrefix
	return nil
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
		h.log.Info(fmt.Sprintf("started http service endpoint ( %s://%s/%s )", h.Scheme, h.ep, h.prefix))
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
