package notify

import (
	"testing"
)

func TestNotifierDoesNotPanic(t *testing.T) {
	n := New(Options{Desktop: false, Bell: false})
	n.Notify("Title", "Message")
	n.Bell()

	n2 := New(Options{Desktop: true, Bell: true})
	n2.Notify("Title", "Message")
	n2.Bell()

	noop := NoopNotifier{}
	noop.Notify("Title", "Message")
	noop.Bell()
}
