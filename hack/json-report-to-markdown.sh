#!/bin/bash

# 
FILENAME=$1
TITLE=$2
OUTPUT=./testing/${FILENAME}.md

# write header
echo "# ${TITLE}" > ${OUTPUT}
echo "" >> ${OUTPUT}
echo "Test | Description | File " >> ${OUTPUT}
echo "---|---|---" >> ${OUTPUT}

# extract
CSV=$(mktemp)
jq -r '.[]|select(.SpecReports!=null)|.SpecReports[]|select(.ContainerHierarchyTexts!=null) | [(.ContainerHierarchyTexts | join("/")), .LeafNodeText, .LeafNodeLocation.FileName] | @tsv ' \
 < ${FILENAME}.json | sort | awk -F'\t' '{ printf "| %s | %s | %s |\n", $1, $2, $3 }'  >> ${OUTPUT}

echo "Report ${TITLE} saved in ${OUTPUT}"