#include "clipboard_darwin.h"

static NSInteger changeCount;

char * getClipboard(int *length) {
  NSPasteboard *pb = [NSPasteboard generalPasteboard];

  if ([pb changeCount] == changeCount) {
    return NULL;
  }

  changeCount = [pb changeCount];

  NSArray *types = @[[NSString class]];
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

int setClipboard(char *buf) {
  return 0;
}
