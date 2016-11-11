#include "clipboard_darwin.h"

static NSInteger changeCount;

char * getClipboard(int *length) {
  NSPasteboard *pb = [NSPasteboard generalPasteboard];

  if ([pb changeCount] == changeCount) {
    return NULL;
  }

  changeCount = [pb changeCount];

  NSString *avail = [pb availableTypeFromArray:@[NSStringPboardType]];
  if (avail == nil) {
    return NULL;
  }

  NSData *data = [pb dataForType:NSStringPboardType];
  if (data == nil) {
    return NULL;
  }

  *length = (int)[data length];
  char *buf = (char *)malloc(*length);
  memcpy(buf, [data bytes], *length);

  return buf;
}

char * setClipboard(char *buf, int length) {
  NSPasteboard *pb = [NSPasteboard generalPasteboard];
  changeCount = [pb declareTypes:@[NSStringPboardType] owner:nil];
  NSData *data = [NSData dataWithBytes:(const void *)buf length:length];

  @try {
    if ([pb setData:data forType:NSStringPboardType] == NO) {
      // This is not a try...catch block just for this exception!
      // [NSPasteboard setData:forType] raises exceptions when there's problems
      // connecting to the pasteboard.
      [NSException raise:@"sshclip error" format:@"Unable to set clipboard data"];
    }
  } @catch (NSException *e) {
    const char *err = [[NSString stringWithFormat:@"%@: %@", [e name], [e reason]] UTF8String];
    int length = strlen(err);
    char *buf = (char *)malloc(length);
    memcpy(buf, err, length);
    return buf;
  } @finally {
    [data release];
  }

  return NULL;
}
