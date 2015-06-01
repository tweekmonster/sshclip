PREFIX = /usr/local
JAIL_USER = sshclip
JAIL_ROOT = /opt/jails/sshclip
CC = gcc
DEST = bin
TARGET = $(DEST)/sshclip-shell
OBJECTS = $(patsubst %.c, %.o, $(wildcard shell/*.c))
HEADERS = $(wildcard src/*.h)
CFLAGS = -Wall -std=gnu99 -O2

.PHONY: default all bindir clean

all: bindir default
debug: CFLAGS += -g -DDEBUG
debug: all

jail: all
jail:
	scripts/makejail $(JAIL_USER) $(JAIL_ROOT)
	

default: $(TARGET)

bindir: $(DEST)

$(DEST):
	mkdir -p $@

%.o: %.c $(HEADERS)
	$(CC) $(CFLAGS) -c $< -o $@

.PRECIOUS: $(TARGET) $(OBJECTS)

$(TARGET): $(OBJECTS)
	$(CC) $(OBJECTS) -Wall -o $@

install:
	install -o 0 -g 0 -m 0755 $(TARGET) $(PREFIX)/bin

clean:
	-rm -f shell/*.o
	-rm -rf $(DEST)

#  vim: set ts=4 sw=4 tw=0 noet :
