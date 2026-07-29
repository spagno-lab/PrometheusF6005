package ont

import (
	"net/http"
)

type Session struct {
	*http.Client
	Endpoint string
}

func (s *Session) getAndClose(url string) {
	resp, err := s.Get(url)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}
