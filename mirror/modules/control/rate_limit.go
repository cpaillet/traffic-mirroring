package control

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/criteo/traffic-mirroring/mirror"
	"github.com/criteo/traffic-mirroring/mirror/expr"
	"github.com/criteo/traffic-mirroring/mirror/registry"
)

const (
	RateLimitName = "control.rate_limit"
)

func init() {
	registry.Register(RateLimitName, NewRateLimit)
}

type RateLimitConfig struct {
	RPS *expr.NumberExpr `json:"rps"`
}

type RateLimit struct {
	ctx *mirror.ModuleContext
	cfg RateLimitConfig
	out chan mirror.Request

	ready    bool
	interval time.Duration
}

func NewRateLimit(ctx *mirror.ModuleContext, cfg []byte) (mirror.Module, error) {
	c := RateLimitConfig{}
	err := json.Unmarshal(cfg, &c)
	if err != nil {
		return nil, err
	}

	mod := &RateLimit{
		ctx: ctx,
		cfg: c,
		out: make(chan mirror.Request),
	}

	return mod, nil
}

func (m *RateLimit) Context() *mirror.ModuleContext {
	return m.ctx
}

func (m *RateLimit) Children() [][]mirror.Module {
	return nil
}

func (m *RateLimit) Output() <-chan mirror.Request {
	return m.out
}

func (m *RateLimit) SetInput(c <-chan mirror.Request) {
	go func() {

		for r := range c {
			if !m.ready {
				err := m.init(r)
				if err != nil {
					log.Fatalf("%s: %s", RateLimitName, err)
				}
			}

			start := time.Now()

			m.ctx.HandledRequest()
			m.out <- r

			time.Sleep(m.interval - time.Since(start))
		}
		close(m.out)
	}()
}

func (m *RateLimit) init(r mirror.Request) error {
	v, err := m.cfg.RPS.EvalFloat(r)
	if err != nil {
		return fmt.Errorf("cannot evaluate rps: %s", err)
	}

	m.ready = true
	m.interval = time.Duration(float64(time.Second) / v)
	return nil
}
