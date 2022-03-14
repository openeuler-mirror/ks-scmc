all:
	make -C rpc all
	mkdir -p build
	cd build && cmake -DCMAKE_INSTALL_PREFIX=/usr .. && make

install:
	cd build && make install

.PHONY: clean
clean:
	make -C rpc clean
	rm -rf build/
