package goba

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
	zero "github.com/wdvxdr1123/ZeroBot"
)

var (
	ErrInvalidFirstContextType = errors.New("invalid first context type")
)

var (
	eventType = reflect.TypeOf(Event{})
	// requestType = reflect.TypeOf(zero.APIRequest{})
)

type generalctx struct {
	mu  sync.Mutex
	ctx []events
}

func addctx[T Event | zero.APIRequest](
	ctx *generalctx, v *T, ctxcap, evcap int,
) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	typ := reflect.TypeOf(v).Elem()

	if len(ctx.ctx) == 0 {
		// must triggered by event
		if !typ.AssignableTo(eventType) {
			panic(errors.Wrap(ErrInvalidFirstContextType, typ.String()))
		}

		// ctxcap & evcap must > 0, no need to check
		evs := make(events, 1, evcap)
		evs[0] = v
		ctx.ctx = make([]events, 1, ctxcap)
		ctx.ctx[0] = evs
		return
	}
	// Get the last events slice
	lastEvents := &ctx.ctx[len(ctx.ctx)-1]

	// Check if the type matches the first element of the last events
	firstElemType := reflect.TypeOf((*lastEvents)[0]).Elem()
	if typ.AssignableTo(firstElemType) {
		// Same type, add to the last events
		if len(*lastEvents) >= evcap {
			// Shift elements forward by 1 (discard first element)
			copy((*lastEvents)[:], (*lastEvents)[1:])
			(*lastEvents)[len(*lastEvents)-1] = any(v)
		} else {
			*lastEvents = append(*lastEvents, any(v))
		}
		return
	}

	// Different type or empty last events, create new events slice
	if len(ctx.ctx) >= ctxcap {
		// Shift elements forward by 2 (user-assistant pair)
		copy(ctx.ctx[:], ctx.ctx[2:])
		ctx.ctx = ctx.ctx[:len(ctx.ctx)-2]
	}
	evs := make(events, 1, evcap)
	evs[0] = v
	ctx.ctx = append(ctx.ctx, evs)
}
