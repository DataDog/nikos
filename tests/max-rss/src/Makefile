ROOT_DIR := ..

include $(ROOT_DIR)/Makefile.inc

MAKE ?= make
CLANG ?= clang
LLVM_STRIP ?= llvm-strip

BPFTOOL=$(TOOLS_DIR)/bpftool

THIRD_PARTY = $(LIBBPF_OBJ)

CLANG_BPF_SYS_INCLUDES = $(shell $(CLANG) -v -E - < /dev/null 2>&1 \
			 | sed -n '/<...> search starts here:/,/End of search list./{ s| \(/.*\)|-idirafter \1|p }')

.PHONY: all clean

MAX_RSS := max_rss
all: $(MAX_RSS)

clean:
	rm -f *.o
	rm -f $(MAX_RSS).skel.h
	rm -f $(MAX_RSS)

$(MAX_RSS).bpf.o: $(MAX_RSS).bpf.c $(LIBBPF_OBJ) $(wildcard %.h)
	$(CLANG) 				\
		-D__BPF_HEADER__ 		\
		-DBPF_NO_PRESERVE_ACCESS_INDEX	\
		-D__TARGET_ARCH_$(ARCH) 	\
		-D__KERNEL__ 			\
		-I$(INC_DIR) 			\
		-g3 -O2 -target bpf  		\
		$(CLANG_BPF_SYS_INCLUDES) 	\
		-c $(filter %.c,$^) 		\
		-o $@

# Generate bpf skeleton
$(MAX_RSS).skel.h: $(MAX_RSS).bpf.o
	$(BPFTOOL) gen skeleton $< > $@


$(MAX_RSS).o: $(MAX_RSS).c $(wildcard %.h) $(MAX_RSS).skel.h
	$(CC) $(CFLAGS) $(INCLUDES) -c $(filter %.c, $^) -o $@

# Build application binary
$(MAX_RSS): $(MAX_RSS).o $(THIRD_PARTY)
	$(CC) -static $(CFLAGS) $(INCLUDES) $(MAX_RSS).o $(THIRD_PARTY) -lelf -lz -o $@

.PHONY: all clean
