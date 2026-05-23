package server

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/andrebq/postigo/internal/ioutil"
	"github.com/hashicorp/yamux"
)

type (
	trafficManager struct {
		dialRequests       chan dialRequest
		registerRequests   chan registerRequest
		unregisterRequests chan unregisterRequest

		done chan signal
	}
	registerRequest struct {
		nodename string
		session  *yamux.Session
		regid    chan uint64
	}
	unregisterRequest struct {
		regid uint64
	}
	dialRequest struct {
		nodename string
		result   chan dialResult
	}
	dialResult struct {
		stream *yamux.Stream
		err    error
	}

	nodelist []node

	node struct {
		name    string
		session *yamux.Session
		load    uint64
		regid   uint64
	}

	signal struct{}
)

func (n node) compare(o node) int {
	if i := strings.Compare(n.name, o.name); i != 0 {
		return i
	} else {
		// lower loads have higher priorty
		// hence are sorted first
		return int(n.load - o.load)
	}
}

func newTrafficManager() *trafficManager {
	return &trafficManager{
		dialRequests:       make(chan dialRequest, 1),
		registerRequests:   make(chan registerRequest, 20),
		unregisterRequests: make(chan unregisterRequest, 100),

		done: make(chan signal),
	}
}

func (tm *trafficManager) route(ctx context.Context) error {
	defer close(tm.done)
	var nodes nodelist
	var nextRegID uint64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case unreg := <-tm.unregisterRequests:
			for i, v := range nodes {
				if v.regid == unreg.regid {
					copy(nodes[i:], nodes[i+1:])
					nodes[len(nodes)-1] = node{}
					nodes = nodes[:len(nodes)-1]
					break
				}
			}
		case reg := <-tm.registerRequests:
			nextRegID++
			nodes = append(nodes, node{
				session: reg.session,
				name:    reg.nodename,
				load:    0,
				regid:   nextRegID,
			})
			reg.regid <- nextRegID
		case req := <-tm.dialRequests:
			found := false
			for i, v := range nodes {
				found = v.name == req.nodename
				if found {
					go tm.dispatch(req, v.session)
					nodes[i].load++
					slices.SortFunc(nodes, node.compare)
					break
				}
			}
			if !found {
				tm.dispatch(req, nil)
			}
		}
	}
}

func (tm *trafficManager) dispatch(req dialRequest, sess *yamux.Session) {
	if sess == nil {
		req.result <- dialResult{
			err: errors.New("not found"),
		}
		return
	}
	stream, err := sess.OpenStream()
	// err should be processed by whoever receives the
	// dialResult not by dispatch
	select {
	case <-tm.done:
		if stream != nil {
			stream.Close()
		}
		close(req.result)
	case req.result <- dialResult{
		stream: stream,
		err:    err,
	}:
	}

}

func (tm *trafficManager) registerNode(nodename string, session *yamux.Session) (uint64, <-chan signal) {
	regreq := registerRequest{
		nodename: nodename,
		session:  session,
		regid:    make(chan uint64, 1),
	}
	select {
	case tm.registerRequests <- regreq:
		select {
		case regid := <-regreq.regid:
			return regid, tm.done
		case <-tm.done:
			return 0, tm.done
		}
	case <-tm.done:
		return 0, tm.done
	}
}

func (tm *trafficManager) unregisterNode(regid uint64) {
	select {
	case tm.unregisterRequests <- unregisterRequest{regid}:
	case <-tm.done:
	}
}

func (tm *trafficManager) dialNode(ctx context.Context, nodename string, stream *yamux.Stream) error {
	dr := dialRequest{
		nodename: nodename,
		result:   make(chan dialResult, 1),
	}
	select {
	case tm.dialRequests <- dr:
	case <-ctx.Done():
		return ctx.Err()
	case <-tm.done:
		return errors.New("closed")
	}
	var res dialResult
	select {
	case res = <-dr.result:
	case <-ctx.Done():
		return ctx.Err()
	case <-tm.done:
		return errors.New("closed")
	}
	if res.err != nil {
		stream.Close()
		return fmt.Errorf("unable to acquire remote stream: %w", res.err)
	}
	errCh := ioutil.BackgroundCopy(stream, res.stream)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		stream.Close()
		res.stream.Close()
		return err
	}
}
