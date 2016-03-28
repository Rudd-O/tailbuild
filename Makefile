tailbuild: tailbuild.go
	go build tailbuild.go

BINDIR=/usr/bin
DESTDIR=

all: tailbuild

clean:
	rm -f tailbuild

dist: clean
	DIR=tailbuild-`awk '/^Version:/ {print $$2}' tailbuild.spec` && FILENAME=$$DIR.tar.gz && tar cvzf "$$FILENAME" --exclude "$$FILENAME" --exclude .git --exclude .gitignore -X .gitignore --transform="s|^|$$DIR/|" --show-transformed *

rpm: dist
	T=`mktemp -d` && rpmbuild --define "_topdir $$T" -ta tailbuild-`awk '/^Version:/ {print $$2}' tailbuild.spec`.tar.gz || { rm -rf "$$T"; exit 1; } && mv "$$T"/RPMS/*/* "$$T"/SRPMS/* . || { rm -rf "$$T"; exit 1; } && rm -rf "$$T"

install: all
	install -Dm 755 tailbuild -t $(DESTDIR)/$(BINDIR)/
