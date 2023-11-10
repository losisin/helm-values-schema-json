#!/usr/bin/env sh

# borrowed from https://github.com/technosophos/helm-template

PROJECT_NAME="helm-values-schema-json"
PROJECT_GH="losisin/$PROJECT_NAME"
HELM_PLUGIN_PATH="$HELM_PLUGIN_DIR"

# Convert the HELM_PLUGIN_PATH to unix if cygpath is
# available. This is the case when using MSYS2 or Cygwin
# on Windows where helm returns a Windows path but we
# need a Unix path
if type cygpath >/dev/null 2>&1; then
  echo Use Sygpath
  HELM_PLUGIN_PATH=$(cygpath -u "$HELM_PLUGIN_PATH")
fi

if [ "$SKIP_BIN_INSTALL" = "1" ]; then
  echo "Skipping binary install"
  exit
fi

# initArch discovers the architecture for this system.
initArch() {
  ARCH=$(uname -m)
  case "$ARCH" in
    armv5*) ARCH="armv5";;
    armv6*) ARCH="armv6";;
    armv7*) ARCH="armv7";;
    aarch64) ARCH="arm64";;
    x86) ARCH="386";;
    x86_64) ARCH="amd64";;
    i686) ARCH="386";;
    i386) ARCH="386";;
  esac
}

# initOS discovers the operating system for this system.
initOS() {
  OS=$(uname | tr '[:upper:]' '[:lower:]')

  case "$OS" in
    # Msys support
    msys*) OS='windows';;
    # Minimalist GNU for Windows
    mingw*) OS='windows';;
    darwin) OS='darwin';;
  esac
}

# verifySupported checks that the os/arch combination is supported for
# binary builds.
verifySupported() {
  supported="linux_arm64\nlinux_amd64\ndarwin_amd64\ndarwin_arm64\nwindows_amd64\nwindows_arm64"
  if ! echo "$supported" | grep -q "${OS}_${ARCH}"; then
    echo "No prebuild binary for ${OS}_${ARCH}."
    exit 1
  fi

  if ! type "curl" >/dev/null 2>&1 && ! type "wget" >/dev/null 2>&1; then
    echo "Either curl or wget is required"
    exit 1
  fi
  echo "Support ${OS}_${ARCH}"
}

# getDownloadURL checks the latest available version.
getDownloadURL() {
  # Determine last tag based on VCS download
  version=$(git describe --tags --abbrev=0 >/dev/null 2>&1)
  # If no version found (because of no git), try fetch from plugin
  if [ -z "$version" ]; then
    echo "No version found"
    version=v$(sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' plugin.yaml)  
  fi

  # Setup Download Url
  DOWNLOAD_URL="https://github.com/${PROJECT_GH}/releases/download/${version}/${PROJECT_NAME}_${version#v}_${OS}_${ARCH}.tgz"
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
downloadFile() {
  PLUGIN_TMP_FOLDER="/tmp/_dist/"
  [ -d "$PLUGIN_TMP_FOLDER" ] && rm -r "$PLUGIN_TMP_FOLDER" >/dev/null
  mkdir -p "$PLUGIN_TMP_FOLDER"
  echo "Downloading $DOWNLOAD_URL to location $PLUGIN_TMP_FOLDER"
  if type "curl" >/dev/null 2>&1; then
      (cd "$PLUGIN_TMP_FOLDER" && curl -LO "$DOWNLOAD_URL")
  elif type "wget" >/dev/null 2>&1; then
      wget -P "$PLUGIN_TMP_FOLDER" "$DOWNLOAD_URL"
  fi
}

# installFile unpacks and installs the file
installFile() {
  cd "/tmp"
  DOWNLOAD_FILE=$(find ./_dist -name "*.tgz")
  HELM_TMP="/tmp/$PROJECT_NAME"
  mkdir -p "$HELM_TMP"
  tar xf "$DOWNLOAD_FILE" -C "$HELM_TMP"
  HELM_TMP_BIN="$HELM_TMP/schema"
  echo "Preparing to install into ${HELM_PLUGIN_PATH}"
  # Use * to also copy the file with the exe suffix on Windows
  cp "$HELM_TMP_BIN"* "$HELM_PLUGIN_PATH"
  rm -r "$HELM_TMP"
  rm -r "$PLUGIN_TMP_FOLDER"
  echo "$PROJECT_NAME installed into $HELM_PLUGIN_PATH"
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    echo "Failed to install $PROJECT_NAME"
    printf "\tFor support, go to https://github.com/%s.\n" "$PROJECT_GH"
  fi
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  # To avoid to keep track of the Windows suffix,
  # call the plugin assuming it is in the PATH
  PATH=$PATH:$HELM_PLUGIN_PATH
  schema -help
}

# Execution

#Stop execution on any error
trap "fail_trap" EXIT
set -e
initArch
initOS
verifySupported
getDownloadURL
downloadFile
installFile
testVersion
