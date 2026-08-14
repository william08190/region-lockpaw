package platform

import "testing"

func TestSecureUnlockFlowIgnoresExternalSessionRoundTrip(t *testing.T) {
	var flow secureUnlockFlow

	if flow.observeSessionLock() {
		t.Fatal("external session lock was treated as app-requested")
	}
	if flow.observeSessionUnlock() {
		t.Fatal("Win+L session round trip authorized region unlock")
	}
}

func TestSecureUnlockFlowAuthorizesRequestedRoundTrip(t *testing.T) {
	var flow secureUnlockFlow
	flow.begin()

	if !flow.observeSessionLock() {
		t.Fatal("app-requested session lock was not recognized")
	}
	if !flow.observeSessionUnlock() {
		t.Fatal("completed secure-sign-in round trip did not authorize unlock")
	}
	if flow.observeSessionUnlock() {
		t.Fatal("secure unlock authorization was reusable")
	}
}

func TestSecureUnlockFlowRequiresObservedSessionLock(t *testing.T) {
	var flow secureUnlockFlow
	flow.begin()

	if flow.observeSessionUnlock() {
		t.Fatal("session unlock without a preceding session lock was authorized")
	}
	if flow.observeSessionLock() {
		t.Fatal("unexpected session unlock did not cancel the pending request")
	}
}

func TestSecureUnlockFlowResetCancelsPendingRequest(t *testing.T) {
	var flow secureUnlockFlow
	flow.begin()
	flow.observeSessionLock()
	flow.reset()

	if flow.observeSessionUnlock() {
		t.Fatal("reset flow still authorized unlock")
	}
}
