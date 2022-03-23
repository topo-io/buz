package manifold

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/silverton-io/honeypot/pkg/config"
	"github.com/silverton-io/honeypot/pkg/envelope"
	"github.com/silverton-io/honeypot/pkg/sink"
	"github.com/silverton-io/honeypot/pkg/tele"
)

type Manifold struct {
	ShutdownChan          *chan bool
	envelopeChan          *chan envelope.Envelope
	bufferRecordThreshold int
	bufferByteThreshold   int
	bufferTimeThreshold   int
	sink                  *sink.Sink
	lastPurged            time.Time
}

func (m *Manifold) initialize(conf config.Manifold, sink *sink.Sink) {
	c := make(chan envelope.Envelope, 10000) // FIXME! Configurable buffer size
	sDown := make(chan bool, 1)
	m.ShutdownChan = &sDown
	m.envelopeChan = &c
	m.bufferRecordThreshold = conf.BufferRecordThreshold
	m.bufferByteThreshold = conf.BufferByteThreshold
	m.bufferTimeThreshold = conf.BufferTimeThreshold
	m.sink = sink
	m.lastPurged = time.Now()
}

func (m Manifold) Enqueue(envelopes []envelope.Envelope) {
	for _, e := range envelopes {
		log.Debug().Msg("enqueing envelope")
		*m.envelopeChan <- e
		// FIXME! Add durability option
	}
}

func BuildManifold(conf config.Manifold, sink *sink.Sink) (manifold *Manifold, err error) {
	log.Debug().Msg("building manifold")
	m := Manifold{}
	m.initialize(conf, sink)
	return &m, nil
}

func Run(manifold *Manifold, meta *tele.Meta) {
	log.Debug().Msg("running manifold")
	go func() {
		ctx := context.Background()
		var invalidEnvelopes []envelope.Envelope
		var validEnvelopes []envelope.Envelope
		sink := *manifold.sink
		for {
			e := <-*manifold.envelopeChan
			if *e.IsValid {
				log.Debug().Msg("appending valid envelope to buffer...")
				validEnvelopes = append(validEnvelopes, e)
				meta.ProtocolStats.IncrementValid(e.EventProtocol, e.EventSchema, 1)
			} else {
				log.Debug().Msg("appending invalid envelope to buffer...")
				invalidEnvelopes = append(invalidEnvelopes, e)
				meta.ProtocolStats.IncrementInvalid(e.EventProtocol, e.EventSchema, 1)
			}
			if len(validEnvelopes) >= manifold.bufferRecordThreshold || len(invalidEnvelopes) >= manifold.bufferRecordThreshold { // FIXME! Break out buffer purge
				log.Debug().Msg("purging envelope buffers")
				sink.BatchPublishValid(ctx, validEnvelopes)
				sink.BatchPublishInvalid(ctx, invalidEnvelopes)
				meta.BufferPurgeStats.Increment()
				manifold.lastPurged = time.Now()
				invalidEnvelopes = nil
				validEnvelopes = nil
			}
		}
	}()
}
