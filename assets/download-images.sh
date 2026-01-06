#!/bin/bash
set -e

list="images.txt"
images="vcluster-images"

usage () {
    echo "USAGE: $0 [--image-list images.txt] [--images vcluster-images]"
    echo "  [-l|--image-list path] text file with list of images; one image per line."
    echo "  [-i|--images path] directory to store OCI image archives (without extension)."
    echo "  [-h|--help] Usage message"
    echo ""
    echo "This script downloads multi-arch images using skopeo, preserving manifest lists."
    echo "Requires: skopeo (https://github.com/containers/skopeo)"
}

while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -i|--images)
        images="$2"
        shift # past argument
        shift # past value
        ;;
        -l|--image-list)
        list="$2"
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

# Check for skopeo
if ! command -v skopeo &> /dev/null; then
    echo "Error: skopeo is required but not installed."
    echo "Install it from: https://github.com/containers/skopeo"
    exit 1
fi

# Create output directory
mkdir -p "${images}"

pulled=0
failed=0

while IFS= read -r i; do
    [ -z "${i}" ] && continue
    [[ "${i}" == "#"* ]] && continue

    # Create a safe filename from the image reference
    # Replace / with _ and : with __ to avoid collisions
    # (e.g., registry.io:5000/image vs registry.io/5000/image)
    safe_name=$(echo "${i}" | sed 's|:|__|g; s|/|_|g')
    archive_path="${images}/${safe_name}"

    echo "Downloading ${i}..."

    # Use skopeo to copy the multi-arch image to an OCI directory
    # --all flag ensures all platforms in the manifest list are copied
    if skopeo copy --all "docker://${i}" "oci:${archive_path}"; then
        echo "Image download success: ${i}"
        ((++pulled))
    else
        echo "Image download failed: ${i}"
        ((++failed))
    fi
done < "${list}"

if [[ ${pulled} -eq 0 ]]; then
    echo "No images were downloaded successfully"
    exit 1
fi

# Copy the image list for reference during push
cp "${list}" "${images}/images.txt"

echo ""
echo "Done. Downloaded ${pulled} images (${failed} failed)"
echo "Images stored in: ${images}/"
echo ""
echo "To create a portable archive, run:"
echo "  tar -czvf ${images}.tar.gz ${images}/"
