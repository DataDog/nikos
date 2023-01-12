set -euxo pipefail

SOURCE_FILES_PATH=$1

NIKOS_BIN_PATH=/opt/nikos/bin

# Build & install binary
go build $SOURCE_FILES_PATH

sudo mkdir -p $NIKOS_BIN_PATH
sudo mv nikos $NIKOS_BIN_PATH
sudo chmod -R 0755 $NIKOS_BIN_PATH