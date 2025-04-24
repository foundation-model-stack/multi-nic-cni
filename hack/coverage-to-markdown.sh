#!/bin/bash
# requires go tool

FILENAME=$1
TITLE=$2
OUTPUT=./testing/${FILENAME}.md

# write header
echo "# ${TITLE}" > ${OUTPUT}
echo "" >> ${OUTPUT}
echo "File | Function | Coverage " >> ${OUTPUT}
echo "---|---|---" >> ${OUTPUT}

# extract
go tool cover -func=${FILENAME} | awk -F'[ \t]+' '{ printf "| %s | %s | %s |\n", $1, $2, $3 }' >> ${OUTPUT}

echo "Report ${TITLE} saved in ${OUTPUT}"