export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y ca-certificates
update-ca-certificates

/opt/nikos $@