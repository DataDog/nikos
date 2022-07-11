ROOT_DIR := ..

include $(ROOT_DIR)/Makefile.inc

LIBBPF_SRC := $(abspath ./libbpf/src)
BPFTOOL_SRC := $(abspath ./bpftool/src)

BPFTOOL := bpftool

all: $(LIBBPF_OBJ) $(BPFTOOL)

clean:
	make -C $(LIBBPF_SRC) clean
	make -C $(BPFTOOL_SRC) clean

$(TOOLS_DIR):
	mkdir $@

$(INC_DIR)/libbpf:
	mkdir -p $@

# Build libbpf
$(LIBBPF_OBJ): $(wildcard $(LIBBPF_SRC)/*.[ch] $(LIBBPF_SRC)/Makefile) | $(INC_DIR)/libbpf
	$(call msg,LIBBPF,$@)
	$(Q)$(MAKE) -C $(LIBBPF_SRC) BUILD_STATIC_ONLY=1		      \
		    OBJDIR=$(dir $@)/libbpf DESTDIR=$(dir $@)		      \
		    INCLUDEDIR= LIBDIR= UAPIDIR=			      \
		    install
# Build bpftool
$(BPFTOOL): | $(TOOLS_DIR)
	$(call msg,BPFTOOL,$@)
	$(Q)$(MAKE) ARCH= CROSS_COMPILE= OUTPUT=$(TOOLS_DIR) -C $(BPFTOOL_SRC)

.PHONY: all clean $(LIBBPF_OBJ) $(BPFTOOL)
