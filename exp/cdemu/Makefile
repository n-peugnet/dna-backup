PKGS := vhba-module-20210418 libmirage-3.2.5 cdemu-daemon-3.2.5 cdemu-client-3.2.5
TARS := $(PKGS:%=%.tar.xz)


.PHONY: install
install: install-cdemu-daemon-3.2.5 install-cdemu-client-3.2.5 install-vhba-module

.PHONY: install-cdemu-client-3.2.5
install-cdemu-client-3.2.5: cdemu-client-3.2.5/install_manifest.txt

.PHONY: install-cdemu-daemon-3.2.5
install-cdemu-daemon-3.2.5: cdemu-daemon-3.2.5/install_manifest.txt install-libmirage-3.2.5
	sudo install -D -m 644 cdemu-daemon-3.2.5/service-example/cdemu-daemon.service /usr/local/lib/systemd/user/cdemu-daemon.service
	sudo sed -i 's#/usr/#/usr/local/#' /usr/local/lib/systemd/user/cdemu-daemon.service
	sudo install -D -m 644 cdemu-daemon-3.2.5/service-example/net.sf.cdemu.CDEmuDaemon.service /usr/local/share/dbus-1/services/net.sf.cdemu.CDEmuDaemon.service

.PHONY: install-libmirage-3.2.5
install-libmirage-3.2.5: libmirage-3.2.5/install_manifest.txt
	sudo ldconfig

.PHONY: install-vhba-module
install-vhba-module: vhba-module-20210418/vhba.ko
	-sudo insmod vhba-module-20210418/vhba.ko
	sudo chown $$USER /dev/vhba_ctl

.PHONY: uninstall
uninstall: uninstall-cdemu-daemon-3.2.5 uninstall-cdemu-client-3.2.5 uninstall-libmirage-3.2.5
	sudo rm -rf /usr/local/lib/systemd/user/cdemu-daemon.service /usr/local/share/dbus-1/services/net.sf.cdemu.CDEmuDaemon.service
	-sudo rmmod vhba

.PHONY: uninstall-%
uninstall-%: %/install_manifest.txt
	sort $< | while read line; do \
		sudo rm -f $$line; \
		sudo rm -d `dirname $$line` 2>&1 && echo success; \
	done | uniq | tac | sed '/success/I,+1 d'
	rm -f $<

.PHONY: clean
clean:
	rm -f $(TARS)
	rm -rf $(PKGS)

cdemu-client-3.2.5/install_manifest.txt: | cdemu-client-3.2.5/src/cdemu
	@echo "Don't worry there is an uninstall target"
	sudo $(MAKE) -C $(@D) install

cdemu-client-3.2.5/src/cdemu: | cdemu-client-3.2.5/Makefile
	$(MAKE) -C $(@D)

cdemu-daemon-3.2.5/install_manifest.txt: | cdemu-daemon-3.2.5/Makefile cdemu-daemon-3.2.5/cdemu-daemon
	@echo "Don't worry there is an uninstall target"
	sudo $(MAKE) -C $(@D) install

cdemu-daemon-3.2.5/cdemu-daemon: | cdemu-daemon-3.2.5/Makefile
	$(MAKE) -C $(@D)

cdemu-daemon-3.2.5/Makefile: | cdemu-daemon-3.2.5 libmirage-3.2.5/install_manifest.txt
	cd $(@D) && cmake .

libmirage-3.2.5/install_manifest.txt: libmirage-3.2.5/libmirage.so
	@echo "Please enter your password. Don't worry there is an uninstall target"
	sudo $(MAKE) -C $(@D) install

libmirage-3.2.5/libmirage.so: | libmirage-3.2.5/Makefile
	$(MAKE) -C $(@D)

%/Makefile: | %
	cd $(@D) && cmake .

vhba-module-20210418/vhba.ko: | vhba-module-20210418
	cd $(@D) && $(MAKE)

$(PKGS): %: %.tar.xz
	tar -xvf $<

%.tar.xz:
	wget -q http://downloads.sourceforge.net/cdemu/$@

