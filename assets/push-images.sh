#!/bin/bash
set -e

images="vcluster-images"
list=""
insecure="false"

usage () {
    echo "USAGE: $0 --registry my.registry.com:5000 [--images vcluster-images]"
    echo "  [-i|--images path] directory containing OCI image archives (created by download-images.sh)."
    echo "  [-l|--image-list path] text file with list of images (optional, uses images/images.txt if not specified)."
    echo "  [-r|--registry registry:port] target private registry:port."
    echo "  [-k|--insecure] skip TLS verification for target registry (for HTTP registries)."
    echo "  [-h|--help] Usage message"
    echo ""
    echo "This script pushes multi-arch images using skopeo, preserving manifest lists."
    echo "Requires: skopeo (https://github.com/containers/skopeo)"
}

while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -r|--registry)
        reg="$2"
        shift # past argument
        shift # past value
        ;;
        -l|--image-list)
        list="$2"
        shift # past argument
        shift # past value
        ;;
        -i|--images)
        images="$2"
        shift # past argument
        shift # past value
        ;;
        -k|--insecure)
        insecure="true"
        shift
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

if [[ -z $reg ]]; then
    echo "Error: --registry is required"
    usage
    exit 1
fi

# Strip http:// or https:// prefix if provided (use -E for portable extended regex)
reg=$(echo "${reg}" | sed -E 's|^https?://||')

# Validate registry is not empty after stripping prefix
if [[ -z $reg ]]; then
    echo "Error: --registry value is empty after stripping protocol prefix"
    exit 1
fi

# Check for skopeo
if ! command -v skopeo &> /dev/null; then
    echo "Error: skopeo is required but not installed."
    echo "Install it from: https://github.com/containers/skopeo"
    exit 1
fi

# If images path is a tar.gz file, extract it first
if [[ "${images}" == *.tar.gz && -f "${images}" ]]; then
    echo "Extracting ${images}..."
    tar -xzf "${images}"
    # Extract to current directory, so use basename without .tar.gz suffix
    images="$(basename "${images}" .tar.gz)"
fi

# Use embedded image list if not specified
if [[ -z "${list}" ]]; then
    list="${images}/images.txt"
fi

if [[ ! -f "${list}" ]]; then
    echo "Error: Image list not found: ${list}"
    exit 1
fi

if [[ ! -d "${images}" ]]; then
    echo "Error: Images directory not found: ${images}"
    exit 1
fi

# Build skopeo flags
skopeo_flags="--all"
if [[ "${insecure}" == "true" ]]; then
    skopeo_flags="${skopeo_flags} --dest-tls-verify=false"
fi

pushed=0
failed=0

while IFS= read -r i; do
    [ -z "${i}" ] && continue
    [[ "${i}" == "#"* ]] && continue

    # Create the safe filename (must match download script)
    # Replace / with _ and : with __ to avoid collisions
    safe_name=$(echo "${i}" | sed 's|:|__|g; s|/|_|g')
    archive_path="${images}/${safe_name}"

    if [[ ! -d "${archive_path}" ]]; then
        echo "Warning: Archive not found for ${i}, skipping..."
        ((++failed))
        continue
    fi

    # Strip known registry prefixes and prepend target registry
    image_name=$(echo "$i" | sed 's|^ghcr\.io/||; s|^registry\.k8s\.io/||; s|^quay\.io/||; s|^docker\.io/||')
    target="${reg}/${image_name}"

    echo "Pushing ${i} -> ${target}..."

    # Use skopeo to copy the multi-arch image from OCI directory to target registry
    # --all flag ensures all platforms in the manifest list are pushed
    if skopeo copy ${skopeo_flags} "oci:${archive_path}" "docker://${target}"; then
        echo "Image push success: ${target}"
        ((++pushed))
    else
        echo "Image push failed: ${target}"
        ((++failed))
    fi
done < "${list}"

if [[ ${pushed} -eq 0 ]]; then
    echo "No images were pushed successfully"
    exit 1
fi

echo ""
echo "Done. Pushed ${pushed} images to ${reg} (${failed} failed)"
