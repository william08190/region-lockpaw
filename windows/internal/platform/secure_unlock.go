package platform

type secureUnlockPhase uint8

const (
	secureUnlockIdle secureUnlockPhase = iota
	secureUnlockRequested
	secureUnlockSessionLocked
)

// secureUnlockFlow distinguishes an app-requested Windows secure-sign-in
// round trip from an unrelated session lock such as Win+L. Only the former may
// remove an active region mask when the session becomes available again.
type secureUnlockFlow struct {
	phase secureUnlockPhase
}

func (flow *secureUnlockFlow) begin() {
	flow.phase = secureUnlockRequested
}

func (flow *secureUnlockFlow) observeSessionLock() bool {
	if flow.phase != secureUnlockRequested {
		return false
	}
	flow.phase = secureUnlockSessionLocked
	return true
}

func (flow *secureUnlockFlow) observeSessionUnlock() bool {
	authorized := flow.phase == secureUnlockSessionLocked
	flow.reset()
	return authorized
}

func (flow *secureUnlockFlow) reset() {
	flow.phase = secureUnlockIdle
}
