#include "clipboard_darwin.h"

static NSPasteboard *pb;
static NSInteger lastChangeCount;

void setup() {
  pb = [NSPasteboard generalPasteboard];
}

char * getClipboard() {
  if ([pb changeCount] == lastChangeCount) {
    return NULL;
  }

  const char *text = [[pb stringForType:NSStringPboardType] UTF8String];
  int len = strlen(text);
  char *buf = (char *)malloc(len);
  strncpy(buf, text, len);
  buf[len] = '\0';
  return buf;
}

int setClipboard(char *buf) {
  return 0;
}
