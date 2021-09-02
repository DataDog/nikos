SOURCE_FILES_PATH=$1
NIKOS_LIBS_PATH=$2

NIKOS_BIN_PATH=/opt/nikos/bin
NIKOS_EMBEDDED_PATH=/opt/nikos/embedded

# Unpack dependencies
sudo mkdir -p $NIKOS_EMBEDDED_PATH
sudo tar -xf $NIKOS_LIBS_PATH -C $NIKOS_EMBEDDED_PATH
sudo chmod -R 0755 $NIKOS_EMBEDDED_PATH

# Build list of linker flags
libs=(dnf gio-2.0 modulemd gobject-2.0 ffi yaml gmodule-2.0 repo glib-2.0 pcre z solvext rpm rpmio bz2 solv gpgme assuan gcrypt gpg-error sqlite3 curl nghttp2 ssl crypto json-c lzma xml2 popt zstd)
for k in "${!libs[@]}"; do
    lib=${libs[$k]}
    libs[$k]="${NIKOS_EMBEDDED_PATH}/lib/lib${lib}.a "
done
linker_flags=${libs[*]}

# Define build flags
export CGO_LDFLAGS_ALLOW="-Wl,--wrap=.*"
export PKG_CONFIG_PATH=$NIKOS_EMBEDDED_PATH/lib/pkgconfig
export CGO_LDFLAGS="-L${NIKOS_EMBEDDED_PATH}/lib ${linker_flags} -static-libstdc++ -pthread -ldl -lm"

# Build & install binary
go build -tags "dnf molecule" $SOURCE_FILES_PATH

sudo mkdir -p $NIKOS_BIN_PATH
sudo mv nikos $NIKOS_BIN_PATH
sudo chmod -R 0755 $NIKOS_BIN_PATH