package liveness

import (
	"context"
	"sync/atomic"
	"time"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = ctrl.Log.WithName("liveness")

type Checker struct {
	restClient rest.Interface
	interval   time.Duration
	dead       atomic.Bool
}

var (
	_ manager.Runnable               = (*Checker)(nil)
	_ manager.LeaderElectionRunnable = (*Checker)(nil)
)

func NewChecker(restClient rest.Interface) *Checker {
	return &Checker{
		restClient: restClient,
		interval:   1 * time.Minute,
		dead:       atomic.Bool{},
	}
}

func (c *Checker) IsAlive() bool {
	return !c.dead.Load()
}

func (c *Checker) Start(ctx context.Context) error {
	log.Info("liveness checker starts")

	tick := time.NewTicker(c.interval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			c.check(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (*Checker) NeedLeaderElection() bool {
	return false
}

func (c *Checker) check(ctx context.Context) {
	result := c.restClient.Get().AbsPath("/livez").Do(ctx)
	if result.Error() != nil {
		log.Info("can't connect to /livez of kubeapi-server")
		c.dead.Store(true)
	}
}
