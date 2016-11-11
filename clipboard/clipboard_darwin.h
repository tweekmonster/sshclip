#ifndef CLIPBOARD_DARWIN_H
#define CLIPBOARD_DARWIN_H
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>

char * getClipboard(int *length);
int setClipboard(char *buf);

#endif /* ifndef CLIPBOARD_DARWIN_H */
