INC_DIR = $(abspath $(ROOT_DIR)/include)
LIB_DIR = $(abspath $(ROOT_DIR)/libs)
BUILD_DIR = $(abspath $(ROOT_DIR)/build)
SRC_DIR = $(abspath $(ROOT_DIR)/src)
TOOLS_DIR := $(abspath $(ROOT_DIR)/tools)/

# Libraries
LIBBPF_OBJ := $(INC_DIR)/libbpf.a

INCLUDES := -I$(INC_DIR) -I$(ROOT_DIR) -I$(SRC_DIR) 
