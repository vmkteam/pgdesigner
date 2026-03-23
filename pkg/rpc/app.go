package rpc

import (
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/vmkteam/zenrpc/v2"
)

const quitGracePeriod = 3 * time.Second

// AppService provides application lifecycle methods.
type AppService struct {
	zenrpc.Service
	quitCh chan struct{}
	mu     sync.Mutex
	timer  *time.Timer
}

// NewAppService creates an AppService that signals quit via the provided channel.
func NewAppService(quitCh chan struct{}) *AppService {
	return &AppService{quitCh: quitCh}
}

// Quit starts a delayed shutdown. If Ping is not called within the grace period, the server exits.
//
// zenrpc
func (s *AppService) Quit() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Already closed.
	select {
	case <-s.quitCh:
		return
	default:
	}

	if s.timer != nil {
		s.timer.Reset(quitGracePeriod)
		return
	}

	s.timer = time.AfterFunc(quitGracePeriod, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		select {
		case <-s.quitCh:
		default:
			close(s.quitCh)
		}
	})
}

// Ping cancels a pending shutdown (e.g. after page reload).
//
// zenrpc
func (s *AppService) Ping() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	return "pong"
}

func vcsVersion() string {
	result := "dev"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return result
	}
	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			result = v.Value
		}
	}
	if len(result) > 8 {
		result = result[:8]
	}
	return result
}

// About returns application metadata.
//
//zenrpc:return AboutInfo
func (s *AppService) About() AboutInfo {
	return AboutInfo{
		Name:        "PgDesigner",
		Description: "Visual PostgreSQL Schema Designer",
		Version:     vcsVersion(),
		GoVersion:   runtime.Version(),
		Target:      "PostgreSQL 18",
		Author:      "Sergey Bykov (sergeyfast)",
		License:     "PolyForm Noncommercial 1.0.0",
		GitHub:      "https://github.com/vmkteam/pgdesigner",
	}
}
