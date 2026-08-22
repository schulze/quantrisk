.PHONY: all clean build install service-config deploy quantriskd quantriskcli

all: quantriskd quantriskcli

quantriskd:
	go build -o quantriskd ./cmd/quantriskd

quantriskcli:
	go build -o quantriskcli ./cmd/quantriskcli

build: quantriskd quantriskcli

install: build
	sudo install -m 755 quantriskd /usr/local/bin/quantriskd
	sudo install -m 755 quantriskcli /usr/local/bin/quantriskcli

service-config:
	@if [ -f srv.service ]; then \
		echo "srv.service already exists"; \
	else \
		cp srv.service.example srv.service; \
		echo "Created srv.service; edit its user, paths, and domains before deploying"; \
	fi

deploy: install
	@test -f srv.service || (echo "Missing srv.service; run 'make service-config' and edit it" >&2; exit 1)
	sudo install -m 644 srv.service /etc/systemd/system/srv.service
	sudo systemctl daemon-reload
	sudo systemctl enable --now srv
	sudo systemctl restart srv

clean:
	rm -f quantriskd quantriskcli
