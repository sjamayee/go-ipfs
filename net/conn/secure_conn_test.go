package conn

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	peer "github.com/jbenet/go-ipfs/peer"

	context "github.com/jbenet/go-ipfs/Godeps/_workspace/src/code.google.com/p/go.net/context"
)

func setupSecureConn(t *testing.T, c Conn) Conn {
	c, ok := c.(*secureConn)
	if ok {
		return c
	}

	// shouldn't happen, because dial + listen already return secure conns.
	s, err := newSecureConn(c.Context(), c, peer.NewPeerstore())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSecureClose(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	c1, c2 := setupConn(t, ctx, "/ip4/127.0.0.1/tcp/1234", "/ip4/127.0.0.1/tcp/2345")

	c1 = setupSecureConn(t, c1)
	c2 = setupSecureConn(t, c2)

	select {
	case <-c1.Done():
		t.Fatal("done before close")
	case <-c2.Done():
		t.Fatal("done before close")
	default:
	}

	c1.Close()

	select {
	case <-c1.Done():
	default:
		t.Fatal("not done after cancel")
	}

	c2.Close()

	select {
	case <-c2.Done():
	default:
		t.Fatal("not done after cancel")
	}

	cancel() // close the listener :P
}

func TestSecureCancel(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	c1, c2 := setupConn(t, ctx, "/ip4/127.0.0.1/tcp/1234", "/ip4/127.0.0.1/tcp/2345")

	c1 = setupSecureConn(t, c1)
	c2 = setupSecureConn(t, c2)

	select {
	case <-c1.Done():
		t.Fatal("done before close")
	case <-c2.Done():
		t.Fatal("done before close")
	default:
	}

	cancel()

	// wait to ensure other goroutines run and close things.
	<-time.After(time.Microsecond * 10)
	// test that cancel called Close.

	select {
	case <-c1.Done():
	default:
		t.Fatal("not done after cancel")
	}

	select {
	case <-c2.Done():
	default:
		t.Fatal("not done after cancel")
	}

}

func TestSecureCloseLeak(t *testing.T) {

	var wg sync.WaitGroup

	runPair := func(p1, p2, num int) {
		a1 := strconv.Itoa(p1)
		a2 := strconv.Itoa(p2)
		ctx, cancel := context.WithCancel(context.Background())
		c1, c2 := setupConn(t, ctx, "/ip4/127.0.0.1/tcp/"+a1, "/ip4/127.0.0.1/tcp/"+a2)

		c1 = setupSecureConn(t, c1)
		c2 = setupSecureConn(t, c2)

		for i := 0; i < num; i++ {
			b1 := []byte("beep")
			c1.Out() <- b1
			b2 := <-c2.In()
			if !bytes.Equal(b1, b2) {
				panic("bytes not equal")
			}

			b2 = []byte("boop")
			c2.Out() <- b2
			b1 = <-c1.In()
			if !bytes.Equal(b1, b2) {
				panic("bytes not equal")
			}

			<-time.After(time.Microsecond * 5)
		}

		cancel() // close the listener
		wg.Done()
	}

	var cons = 20
	var msgs = 100
	fmt.Printf("Running %d connections * %d msgs.\n", cons, msgs)
	for i := 0; i < cons; i++ {
		wg.Add(1)
		go runPair(2000+i, 2001+i, msgs)
	}

	fmt.Printf("Waiting...\n")
	wg.Wait()
	// done!

	<-time.After(time.Microsecond * 100)
	if runtime.NumGoroutine() > 10 {
		// panic("uncomment me to debug")
		t.Fatal("leaking goroutines:", runtime.NumGoroutine())
	}
}