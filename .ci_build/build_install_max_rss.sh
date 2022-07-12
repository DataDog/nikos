set -euxo pipefail

RSS_SOURCE_PATH="${1}/tests/max-rss"
RSS_BIN_PATH=/opt/nikos

sudo apt update
sudo apt install libelf-dev

sudo mkdir -p $RSS_BIN_PATH

make -C $RSS_SOURCE_PATH all
sudo mv $RSS_SOURCE_PATH/build/max_rss $RSS_BIN_PATH
sudo chmod 0755 $RSS_BIN_PATH/max_rss
