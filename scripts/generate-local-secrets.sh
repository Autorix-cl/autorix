#!/usr/bin/env sh
# Generate the ignored local Compose environment file without printing secrets.
set -eu

target="${1:-.env}"
if [ -e "$target" ]; then
  echo "refusing to overwrite existing $target" >&2
  exit 1
fi

umask 077
if command -v openssl >/dev/null 2>&1; then
  password="$(openssl rand -hex 32)"
else
  password="$(od -An -N 32 -tx1 /dev/urandom | tr -d ' \n')"
fi

cat >"$target" <<EOF
# Generated locally; never commit this file.
POSTGRES_PASSWORD=$password
EOF
chmod 600 "$target"
echo "created $target with owner-only permissions"
