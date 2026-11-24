#!/usr/bin/env sh
# Generate the ignored local Compose environment file without printing secrets.
set -eu

target="${1:-.env}"
if [ -e "$target" ]; then
  echo "refusing to overwrite existing $target" >&2
  exit 1
fi

umask 077
rand_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    od -An -N 32 -tx1 /dev/urandom | tr -d ' \n'
  fi
}

password="$(rand_hex)"
internal_auth_secret="$(rand_hex)"
identity_signing_secret="$(rand_hex)"

cat >"$target" <<EOF
# Generated locally; never commit this file.
POSTGRES_PASSWORD=$password
AUTORIX_INTERNAL_AUTH_SECRET=$internal_auth_secret
AUTORIX_IDENTITY_SIGNING_SECRET=$identity_signing_secret
EOF
chmod 600 "$target"
echo "created $target with owner-only permissions"
