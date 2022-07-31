package comparer

import (
	"context"
	"runtime"
	"sync"

	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

/*
Currently One2ManyResponseComparer is wrapper around
DualResponseComparer . There is not a lot of performance tradeoff
but can improved in headers,headervalue,cookies comparison

Since it is a wrap around DualResponseComparer . Comparison can be
accelerated using goroutines

*/

// One2ManyResposeComparer : Ideal For testing many vulnerabilites
// Ex: Hidden Parameters or any fuzzings
type One2ManyResponseComparer struct {
	// This is concurrent comparison and uses goroutine to speed up time
	Original *rawhttp.RawHttpResponse
	Many     []*rawhttp.RawHttpResponse
	Ignore   map[Factor]bool /* These Factors are Ignored and are not calculated
	By default all Factors are considered except HeaderValue*/
	Concurrency int
}

type One2ManyResults struct {
	Resp    *rawhttp.RawHttpResponse
	Changes []Change
}

func (c *One2ManyResponseComparer) Compare(ctx context.Context) []One2ManyResults {

	type Sender struct {
		Orig   *rawhttp.RawHttpResponse
		New    *rawhttp.RawHttpResponse
		Ignore map[Factor]bool
	}

	results := []One2ManyResults{}

	var senderch chan Sender = make(chan Sender, len(c.Many))
	var recv chan One2ManyResults = make(chan One2ManyResults)

	wg := &sync.WaitGroup{}
	rwg := &sync.WaitGroup{}

	// First launch reciever
	rwg.Add(1)
	go func(rch <-chan One2ManyResults, ctx context.Context) {
		defer rwg.Done()

		for {
			select {
			case <-ctx.Done():
				return

			case val, ok := <-rch:
				if !ok {
					return
				}
				results = append(results, val)
			}
		}

	}(recv, ctx)

	//create worker thread
	worker := func(sch <-chan Sender, recv chan<- One2ManyResults, ctx context.Context) {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return

			case val, ok := <-sch:
				if !ok {
					return
				}
				d := NewDualResponseComparer(val.Orig, val.New)
				d.Ignore = val.Ignore
				res, _ := d.Compare()
				if len(res) > 0 {
					recv <- One2ManyResults{
						Resp:    val.New,
						Changes: res,
					}
				}
			}
		}

	}

	// launch threads based on concurrency
	for i := 0; i < c.Concurrency; i++ {
		wg.Add(1)
		go worker(senderch, recv, ctx)
	}

	for _, v := range c.Many {
		senderch <- Sender{
			Orig:   c.Original,
			New:    v,
			Ignore: c.Ignore,
		}
	}

	close(senderch)
	wg.Wait()
	close(recv)
	rwg.Wait()

	return results
}

func NewOne2ManyResponseComparer(original *rawhttp.RawHttpResponse, many ...*rawhttp.RawHttpResponse) *One2ManyResponseComparer {
	return &One2ManyResponseComparer{
		Original:    original,
		Many:        many,
		Ignore:      map[Factor]bool{HeaderValue: true},
		Concurrency: runtime.NumCPU(),
	}
}
