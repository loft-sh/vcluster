#!/bin/bash
image_list="images.txt"
images="vcluster-images.tar.gz"
chart_list=""
chart_dir="vcluster-charts"

usage () {
    echo "USAGE: $0 [--image-list images.txt] [--images vcluster-images.tar.gz] [--chart-list charts.txt] [--chart-dir ./vcluster-charts]"
    echo "  [-l|--image-list path] text file with list of images; one image per line."
    echo "  [-i|--images path] tar.gz generated by docker save."
    echo "  [-c|--chart-list path] text file with list of charts; one chart per line."
    echo "  [-cd|--chart-dir path] directory where chart tar.gz files are saved."
    echo "  [-h|--help] Usage message"
}

POSITIONAL=()
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -i|--images)
        images="$2"
        shift # past argument
        shift # past value
        ;;
        -l|--image-list)
        image_list="$2"
        shift # past argument
        shift # past value
        ;;
        -c|--chart-list)
        chart_list="$2"
        shift # past argument
        shift # past value
        ;;
        -cd|--chart-dir)
        chart_dir="$2"
        shift # past argument
        shift # past value
        ;;
        -h|--help)
        help="true"
        shift
        ;;
        *)
        usage
        exit 1
        ;;
    esac
done

if [[ $help ]]; then
    usage
    exit 0
fi

if [ -f "${image_list}" ]; then
    pulled=""
    while IFS= read -r i; do
        [ -z "${i}" ] && continue
        if docker pull "${i}" > /dev/null 2>&1; then
            echo "Image pull success: ${i}"
            pulled="${pulled} ${i}"
        else
            if docker inspect "${i}" > /dev/null 2>&1; then
                pulled="${pulled} ${i}"
            else
                echo "Image pull failed: ${i}"
            fi
        fi
    done < "${image_list}"

    echo "Creating ${images} with $(echo ${pulled} | wc -w | tr -d '[:space:]') images"
    docker save $(echo ${pulled}) | gzip --stdout > ${images}
fi

if [ -n "${chart_list}" ] && [ -f "${chart_list}" ]; then
    mkdir -p "${chart_dir}"
    pulled_charts=""
    while IFS= read -r i; do
        [ -z "${i}" ] && continue
        chart_info=($i)
        chart_url="${chart_info[0]}"
        chart_version="${chart_info[1]}"
        if helm pull "${chart_url}" --version "${chart_version}" --destination "${chart_dir}" > /dev/null 2>&1; then
            echo "Chart pull success: ${chart_url} ${chart_version}"
            pulled_charts="${pulled_charts} ${chart_info}"
        else
            echo "Chart pull failed: ${chart_url} ${chart_version}"
        fi
    done < "${chart_list}"
    echo "Created ${chart_dir} with $(echo ${pulled_charts} | wc -w | tr -d '[:space:]') charts"
fi
