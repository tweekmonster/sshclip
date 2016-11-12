package platform

var notificationEnabled = false

func init() {
	notificationEnabled = setupNotifications()
}

func NotificationsEnabled() bool {
	return notificationEnabled
}

func PostNotification(text string) bool {
	return postNotification(text)
}
