package frontend

// This could be part of package.json
//go:generate sh -c "npx capnp-es -ots ../backend/cpnp/*.capnp"
//go:generate sh -c "mv ../backend/cpnp/*.ts ./src/lib/cpnp/"
