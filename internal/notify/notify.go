package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type Notifier interface {
	Notify(title, message string)
	Bell()
}

type Options struct {
	Desktop bool
	Bell    bool
}

type notifier struct {
	desktop bool
	bell    bool
}

func New(opts Options) Notifier {
	return &notifier{
		desktop: opts.Desktop,
		bell:    opts.Bell,
	}
}

func (n *notifier) Bell() {
	if !n.bell {
		return
	}
	// Best-effort terminal bell (\a / ASCII 7)
	_, _ = fmt.Fprint(os.Stderr, "\a")
}

func (n *notifier) Notify(title, message string) {
	if n.bell {
		n.Bell()
	}
	if !n.desktop {
		return
	}

	// Always run asynchronously so it never blocks the Bubble Tea event loop
	go func() {
		defer func() {
			_ = recover()
		}()

		switch runtime.GOOS {
		case "linux":
			cmd := exec.Command("notify-send", "-a", "Breathing TUI", title, message)
			_ = cmd.Run()
		case "darwin":
			script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
			cmd := exec.Command("osascript", "-e", script)
			_ = cmd.Run()
		case "windows":
			psCmd := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; $template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); $textNodes = $template.GetElementsByTagName("text"); $textNodes.Item(0).AppendChild($template.CreateTextNode("%s")) > $null; $textNodes.Item(1).AppendChild($template.CreateTextNode("%s")) > $null; $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Breathing TUI"); $notification = [Windows.UI.Notifications.ToastNotification]::new($template); $notifier.Show($notification)`, title, message)
			cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
			_ = cmd.Run()
		}
	}()
}

type NoopNotifier struct{}

func (NoopNotifier) Notify(string, string) {}
func (NoopNotifier) Bell()                 {}
