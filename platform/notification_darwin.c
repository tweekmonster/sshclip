#include "notification_darwin.h"
#include <objc/runtime.h>
#include <objc/message.h>

NSString * __swizzled_bundleIdentifier(id self, SEL _cmd)
{
  if (self == [NSBundle mainBundle]) {
    return @"io.esdf.sshclip";
  } else {
    return [self performSelector:@selector(swizzled_bundleIdentifier)];
  }
}

static inline BOOL swizzle(Class cls, SEL orig, SEL replacement, IMP method) {
  BOOL ret = class_addMethod(cls, replacement, method, "@@:");
  if (!ret) {
    return NO;
  }
  method_exchangeImplementations(class_getInstanceMethod(cls, orig),
                                 class_getInstanceMethod(cls, replacement));
  return YES;
}

void setup() {
  // Can't post notifications without a bundle identifier.  Can't get a bundle
  // identifier without jumping through hoops.  Notifications could've been
  // accomplished with `osascript`, but that would've been no fun.
  swizzle([NSBundle class], @selector(bundleIdentifier),
          @selector(swizzled_bundleIdentifier),
          (IMP)__swizzled_bundleIdentifier);
}

int postNotification(char *buf) {
  NSString *text = [NSString stringWithUTF8String:buf];
  NSUserNotification *notification = [[NSUserNotification alloc] init];
  [notification setTitle:@"sshclip"];
  [notification setInformativeText:text];

  NSUserNotificationCenter *center = [NSUserNotificationCenter defaultUserNotificationCenter];
  [center deliverNotification:notification];

  [notification release];
  return 0;
}
