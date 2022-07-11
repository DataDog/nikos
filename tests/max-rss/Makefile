ROOT_DIR := .

include $(ROOT_DIR)/Makefile.inc

all: src

CLANG := $(abspath $(TOOLS_DIR)/clang-14)

clean:
	make -C $(LIB_DIR) clean
	rm -r $(TOOLS_DIR)/*
	rm -r $(INC_DIR)/*
	rm $(BUILD_DIR)/*

src: libs
	make -C $(SRC_DIR) all
	mv $(SRC_DIR)/max_rss $(BUILD_DIR)
libs: download_clang
	make -C $(LIB_DIR) all

download_clang:
	$(shell ./scripts/download-clang.sh)

.PHONY: all libs
