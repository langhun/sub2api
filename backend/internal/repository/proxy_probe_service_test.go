package repository

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ProxyProbeServiceSuite struct {
	suite.Suite
	ctx               context.Context
	proxySrv          *httptest.Server
	prober            *proxyProbeService
	originalProbeURLs []proxyProbeURL
}

func (s *ProxyProbeServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.originalProbeURLs = append([]proxyProbeURL(nil), probeURLs...)
	s.prober = &proxyProbeService{
		allowPrivateHosts: true,
	}
}

func (s *ProxyProbeServiceSuite) TearDownTest() {
	probeURLs = append([]proxyProbeURL(nil), s.originalProbeURLs...)
	if s.proxySrv != nil {
		s.proxySrv.Close()
		s.proxySrv = nil
	}
}

func (s *ProxyProbeServiceSuite) setupProxyServer(handler http.HandlerFunc) {
	s.proxySrv = newLocalTestServer(s.T(), handler)
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_InvalidProxyURL() {
	_, _, err := s.prober.ProbeProxy(s.ctx, "://bad")
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "failed to create proxy client")
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_UnsupportedProxyScheme() {
	_, _, err := s.prober.ProbeProxy(s.ctx, "ftp://127.0.0.1:1")
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "failed to create proxy client")
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_Success_IPAPI() {
	probeURLs = []proxyProbeURL{
		{url: "http://probe.test/trace", parser: "cf-trace"},
	}
	s.setupProxyServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "probe.test/trace") {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "ip=1.2.3.4\nloc=CC\n")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	info, latencyMs, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.NoError(s.T(), err, "ProbeProxy")
	require.GreaterOrEqual(s.T(), latencyMs, int64(0), "unexpected latency")
	require.Equal(s.T(), "1.2.3.4", info.IP)
	require.Equal(s.T(), "CC", info.CountryCode)
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_Success_HTTPBinFallback() {
	probeURLs = []proxyProbeURL{
		{url: "http://probe.test/primary", parser: "cf-trace"},
		{url: "http://probe.test/fallback", parser: "cf-trace"},
	}
	s.setupProxyServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "probe.test/primary") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if strings.Contains(r.RequestURI, "probe.test/fallback") {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "ip=5.6.7.8\nloc=US\n")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	info, latencyMs, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.NoError(s.T(), err, "ProbeProxy should fallback to second trace URL")
	require.GreaterOrEqual(s.T(), latencyMs, int64(0), "unexpected latency")
	require.Equal(s.T(), "5.6.7.8", info.IP)
	require.Equal(s.T(), "US", info.CountryCode)
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_AllFailed() {
	s.setupProxyServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	_, _, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "all probe URLs failed")
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_InvalidJSON() {
	probeURLs = []proxyProbeURL{
		{url: "http://probe.test/primary", parser: "cf-trace"},
		{url: "http://probe.test/fallback", parser: "cf-trace"},
	}
	s.setupProxyServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "probe.test/primary") {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "not-json")
			return
		}
		if strings.Contains(r.RequestURI, "probe.test/fallback") {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "not-json")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	_, _, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "all probe URLs failed")
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_ProxyServerClosed() {
	s.setupProxyServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	s.proxySrv.Close()

	_, _, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.Error(s.T(), err, "expected error when proxy server is closed")
}

func (s *ProxyProbeServiceSuite) TestParseIPAPI_Success() {
	body := []byte(`{"status":"success","query":"1.2.3.4","city":"Beijing","regionName":"Beijing","country":"China","countryCode":"CN"}`)
	info, latencyMs, err := s.prober.parseIPAPI(body, 100)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(100), latencyMs)
	require.Equal(s.T(), "1.2.3.4", info.IP)
	require.Equal(s.T(), "Beijing", info.City)
	require.Equal(s.T(), "Beijing", info.Region)
	require.Equal(s.T(), "China", info.Country)
	require.Equal(s.T(), "CN", info.CountryCode)
}

func (s *ProxyProbeServiceSuite) TestParseIPAPI_Failure() {
	body := []byte(`{"status":"fail","message":"rate limited"}`)
	_, _, err := s.prober.parseIPAPI(body, 100)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "rate limited")
}

func (s *ProxyProbeServiceSuite) TestParseHTTPBin_Success() {
	body := []byte(`{"origin": "9.8.7.6"}`)
	info, latencyMs, err := s.prober.parseHTTPBin(body, 50)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "proxy probe country is unknown")
	require.Equal(s.T(), int64(50), latencyMs)
	require.Equal(s.T(), "9.8.7.6", info.IP)
}

func (s *ProxyProbeServiceSuite) TestParseCFTrace_Success() {
	body := []byte("fl=29f32\nip=9.8.7.6\nloc=JP\ntls=TLSv1.3\n")
	info, latencyMs, err := s.prober.parseCFTrace(body, 50)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(50), latencyMs)
	require.Equal(s.T(), "9.8.7.6", info.IP)
	require.Equal(s.T(), "JP", info.CountryCode)
}

func (s *ProxyProbeServiceSuite) TestParseHTTPBin_NoIP() {
	body := []byte(`{"origin": ""}`)
	_, _, err := s.prober.parseHTTPBin(body, 50)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "no IP found")
}

func TestProxyProbeServiceSuite(t *testing.T) {
	suite.Run(t, new(ProxyProbeServiceSuite))
}
