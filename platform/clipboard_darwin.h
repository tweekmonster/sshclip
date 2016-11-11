#ifndef CLIPBOARD_DARWIN_H
#define CLIPBOARD_DARWIN_H
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>

char * getClipboard(int *length);
char * setClipboard(char *buf, int length);

#endif /* ifndef CLIPBOARD_DARWIN_H */
