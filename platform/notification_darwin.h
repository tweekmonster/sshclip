#ifndef NOTIFICATION_DARWIN
#define NOTIFICATION_DARWIN
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>

void setup();
int postNotification(char *buf);

@interface SSHClipNotification : NSObject <NSUserNotificationCenterDelegate>
- (BOOL)send:(NSString *)text;
@end

#endif /* ifndef NOTIFICATION_DARWIN */
