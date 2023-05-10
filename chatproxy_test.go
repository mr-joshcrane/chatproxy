package chatproxy_test

import (
	"github.com/mr-joshcrane/chatproxy"
	"testing"
)

func TestStart(t *testing.T) {
	t.Parallel()
	chatproxy.Start()
}
