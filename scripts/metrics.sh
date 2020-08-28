#!/usr/bin/env bash
PROM_BIN=/usr/local/bin/prometheus
PROM_DIR=prometheus-2.16.0.darwin-amd64
PROM_TAR=${PROM_DIR}.tar.gz
if [[ -f "$PROM_BIN" ]];
  then
     echo "$PROM_BIN exist"
     prometheus
  else
     curl -L https://github.com/prometheus/prometheus/releases/download/v2.16.0/${PROM_TAR} --output ${PROM_TAR}
     tar zxvf ${PROM_TAR}
     cp ${PROM_DIR}/prometheus /usr/local/bin/prometheus
     rm -rf ${PROM_TAR}
     rm -rf ${PROM_DIR}
     prometheus
fi