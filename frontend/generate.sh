#!/bin/sh
# Generate capnp files
npx capnp-es -ots ../backend/cpnp/*.capnp

# Swap the $ to 'cpnp'
# This should just be a simple find and replace.
# Hopefully this doesn't break stuff.
sed -i 's/\$/cpnp/g' ../backend/cpnp/*.ts

# Move the files to frontend
mv ../backend/cpnp/*.ts ./src/lib/cpnp/
