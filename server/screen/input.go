package screen

import (
	"encoding/json"
	"log"
	"time"
)

const (
	inputRateLimit = 200 // sustained events per second, per viewer
	inputBurst     = 60
)

// limiter is a small token bucket that keeps a misbehaving client from flooding
// the host with injected events.
type limiter struct {
	tokens float64
	last   time.Time
}

func (l *limiter) allow(now time.Time) bool {
	if l.last.IsZero() {
		l.tokens = inputBurst
	} else {
		l.tokens += now.Sub(l.last).Seconds() * inputRateLimit
	}
	l.last = now
	if l.tokens > inputBurst {
		l.tokens = inputBurst
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

// geometry is a snapshot of the mapping inputs for one input event.
type geometry struct {
	d          Display
	imgW, imgH int
	ok         bool
}

func (m *Manager) geometryLocked() geometry {
	if len(m.displays) == 0 || m.imgW <= 0 || m.imgH <= 0 {
		return geometry{}
	}
	return geometry{d: m.displays[0], imgW: m.imgW, imgH: m.imgH, ok: true}
}

// handleBinary dispatches one input message. Injection happens outside the
// manager lock so a slow host call cannot stall frame publishing.
func (m *Manager) handleBinary(v *viewer, data []byte) {
	typ, ok := eventType(data)
	if !ok {
		return
	}
	if typ == msgReleaseAll {
		m.releaseControlFor(v, "client released all")
		return
	}

	m.mu.Lock()
	holder := m.holder == v
	now := time.Now()
	if holder {
		if !v.limiter.allow(now) {
			m.mu.Unlock()
			return
		}
		m.lastInput = now
	}
	geo := m.geometryLocked()
	allowed := m.cfg.AllowControl
	m.mu.Unlock()

	// The lock check comes first so a viewer that lost control to a lock screen
	// hears the real reason instead of "take control first".
	if hostLocked() {
		m.denyLocked(v)
		return
	}
	if !holder {
		m.denyInput(v)
		return
	}
	if !allowed || !geo.ok {
		return
	}

	switch typ {
	case msgPointer:
		ev, err := decodePointer(data)
		if err != nil {
			return
		}
		m.injectPointer(v, geo, ev)
	case msgKey:
		ev, err := decodeKey(data)
		if err != nil {
			return
		}
		if err := m.backend.key(ev.Code, ev.Down); err != nil {
			m.inputError(v, err)
		}
	case msgText:
		text, err := decodeText(data)
		if err != nil || text == "" {
			return
		}
		if err := m.backend.text(text); err != nil {
			m.inputError(v, err)
		}
	}
}

func (m *Manager) injectPointer(v *viewer, geo geometry, ev pointerEvent) {
	x, y := imageToHost(geo.d, geo.imgW, geo.imgH, ev.X, ev.Y)
	var err error
	switch ev.Action {
	case actionMove:
		err = m.backend.movePointer(x, y)
	case actionDown:
		if err = m.backend.movePointer(x, y); err == nil {
			err = m.backend.button(ev.Button, true)
		}
	case actionUp:
		if err = m.backend.movePointer(x, y); err == nil {
			err = m.backend.button(ev.Button, false)
		}
	case actionWheel:
		err = m.backend.wheel(0, ev.Aux)
	default:
		return
	}
	if err != nil {
		m.inputError(v, err)
	}
}

// denyInput tells a non-controlling viewer that its events were dropped. The
// notice is throttled so a stuck client cannot flood the socket.
func (m *Manager) denyInput(v *viewer) {
	m.mu.Lock()
	now := time.Now()
	notify := now.Sub(v.lastDeny) > time.Second
	if notify {
		v.lastDeny = now
	}
	m.mu.Unlock()
	if !notify {
		return
	}
	log.Printf("[screen] input_blocked viewer=%s", v.id)
	payload, _ := json.Marshal(map[string]any{
		"type":    "error",
		"code":    "not_controller",
		"message": "take control before sending input",
	})
	v.enqueue(outMessage{text: true, data: payload})
}

// denyLocked tells a viewer why its input stopped, at most once a second.
func (m *Manager) denyLocked(v *viewer) {
	m.mu.Lock()
	now := time.Now()
	notify := now.Sub(v.lastDeny) > time.Second
	if notify {
		v.lastDeny = now
	}
	m.mu.Unlock()
	if !notify {
		return
	}
	log.Printf("[screen] input_blocked viewer=%s reason=host_locked", v.id)
	payload, _ := json.Marshal(map[string]any{
		"type":    "error",
		"code":    "host_locked",
		"message": "the host screen is locked; unlock it to control this desktop",
	})
	v.enqueue(outMessage{text: true, data: payload})
}

func (m *Manager) inputError(v *viewer, err error) {
	log.Printf("[screen] input_error viewer=%s err=%v", v.id, err)
	payload, _ := json.Marshal(map[string]any{
		"type":    "error",
		"code":    "input_failed",
		"message": err.Error(),
	})
	v.enqueue(outMessage{text: true, data: payload})
}

func (m *Manager) releaseControlFor(v *viewer, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.holder == v {
		m.releaseControlLocked(reason)
	}
}
